package server

import (
	"fmt"

	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/labstack/echo/v5"

	contactmodule "github.com/dugble/dugble/server/internal/audience/contacts"
	contactpropertymodule "github.com/dugble/dugble/server/internal/audience/properties"
	segmentmodule "github.com/dugble/dugble/server/internal/audience/segments"
	broadcastmodule "github.com/dugble/dugble/server/internal/campaigns/broadcasts"
	smscampaignmodule "github.com/dugble/dugble/server/internal/campaigns/campaigns"
	platformbilling "github.com/dugble/dugble/server/internal/commercial/charges/usage"
	planmodule "github.com/dugble/dugble/server/internal/commercial/plans"
	subscriptionmodule "github.com/dugble/dugble/server/internal/commercial/subscriptions"
	walletmodule "github.com/dugble/dugble/server/internal/commercial/wallet"
	dbsqlc "github.com/dugble/dugble/server/internal/database/sqlc"
	emaildelivery "github.com/dugble/dugble/server/internal/delivery/email/outbound"
	smsdelivery "github.com/dugble/dugble/server/internal/delivery/sms/outbound"
	authmodule "github.com/dugble/dugble/server/internal/identity/auth"
	mfamodule "github.com/dugble/dugble/server/internal/identity/mfa"
	sessionmodule "github.com/dugble/dugble/server/internal/identity/sessions"
	usermodule "github.com/dugble/dugble/server/internal/identity/users"
	"github.com/dugble/dugble/server/internal/integrations/dns/netdns"
	domainmodule "github.com/dugble/dugble/server/internal/messaging/domains"
	domainclaimmodule "github.com/dugble/dugble/server/internal/messaging/domains/claims"
	emailmodule "github.com/dugble/dugble/server/internal/messaging/email"
	"github.com/dugble/dugble/server/internal/messaging/email/systemmail"
	emailtenant "github.com/dugble/dugble/server/internal/messaging/email/tenants"
	senderidmodule "github.com/dugble/dugble/server/internal/messaging/senderids"
	smsmodule "github.com/dugble/dugble/server/internal/messaging/sms"
	suppressionmodule "github.com/dugble/dugble/server/internal/messaging/suppressions"
	messagetemplatemodule "github.com/dugble/dugble/server/internal/messaging/templates"
	topicmodule "github.com/dugble/dugble/server/internal/messaging/topics"
	"github.com/dugble/dugble/server/internal/modules/audit"
	auditeventmodule "github.com/dugble/dugble/server/internal/modules/audit/events"
	webhooksmodule "github.com/dugble/dugble/server/internal/modules/webhooks"
	platformwebhook "github.com/dugble/dugble/server/internal/modules/webhooks/events"
	httpmiddleware "github.com/dugble/dugble/server/internal/runtime/middleware"
	"github.com/dugble/dugble/server/internal/security"
	"github.com/dugble/dugble/server/internal/security/authz"
	teammodule "github.com/dugble/dugble/server/internal/tenancy/teams"
	teamtokenmodule "github.com/dugble/dugble/server/internal/tenancy/tokens"
)

type serverMiddleware struct {
	auth         echo.MiddlewareFunc
	csrf         echo.MiddlewareFunc
	tenant       func(authz.Permission) echo.MiddlewareFunc
	tenantAccess func(authz.Permission) echo.MiddlewareFunc
}

func (registry *Registry) registerModules(router *echo.Echo) error {
	cfg := registry.config
	db := registry.postgres
	queries := dbsqlc.New(db)

	renderer, err := systemmail.NewRenderer()
	if err != nil {
		return fmt.Errorf("initialize system email renderer: %w", err)
	}
	notificationEmailService := systemmail.NewEmailService(newSystemEmailQueue(cfg, registry.outbox), renderer, cfg.FrontendURL, cfg.AWS.FromEmail)
	if registry.hubtelPayments != nil {
		registry.hubtelPayments.WithNotifier(notificationEmailService)
	}

	auditRepository := audit.NewRepository(db)
	audit.SetSink(auditRepository)
	sessionRepository := sessionmodule.NewRepository(db)
	authRepository := authmodule.NewRepository(db)
	mfaCipher, err := security.NewSecretCipherKeyring(cfg.EncryptionKeys)
	if err != nil {
		return fmt.Errorf("initialize MFA cipher: %w", err)
	}
	mfaService := mfamodule.NewService(db, mfamodule.NewRepository(queries), mfaCipher, "Dugble").WithNotifier(notificationEmailService)
	authService := authmodule.NewService(authRepository, sessionRepository, notificationEmailService, mfaService)
	userRepository := usermodule.NewRepository(db)
	mfaService.WithRecipientStore(userRepository)
	teamRepository := teammodule.NewRepository(db)
	teamService := teammodule.NewService(teamRepository, notificationEmailService).WithRecipientStore(userRepository)
	teamTokenRepository := teamtokenmodule.NewRepository(db)

	contactRepository := contactmodule.NewRepository(db)
	contactPropertyRepository := contactpropertymodule.NewRepository(db)
	segmentRepository := segmentmodule.NewRepository(db)
	topicRepository := topicmodule.NewRepository(db)
	suppressionRepository := suppressionmodule.NewRepository(db)
	messageTemplateRepository := messagetemplatemodule.NewRepository(queries)
	broadcastRepository := broadcastmodule.NewRepository(db)
	domainRepository := domainmodule.NewRepository(db)
	emailTenantService := emailtenant.NewService(db, emailtenant.NewRepository(db), emailtenant.NewProvisioningQueue(registry.outbox))
	senderIDRepository := senderidmodule.NewRepository(db)
	webhookRepository := webhooksmodule.NewRepository(db)
	webhookEmitter := platformwebhook.NewEmitter(webhookRepository)
	smsRepository := smsmodule.NewRepositoryWithWebhookEmitter(db, webhookEmitter)
	smsCampaignRepository := smscampaignmodule.NewRepository(db)
	billingService := platformbilling.NewService(platformbilling.NewRepository(db)).WithBalanceNotifier(notificationEmailService)
	smsService := smsmodule.NewService(smsRepository, registry.smsSender, smsdelivery.NewQueue(registry.outbox), billingService)
	emailAPIService := emailmodule.NewService(
		emailmodule.NewRepository(db),
		emaildelivery.NewQueue(registry.outbox),
		emailmodule.ServiceConfig{DefaultFromEmail: cfg.AWS.FromEmail, DefaultProvider: domainmodule.DefaultProvider, DefaultRegion: cfg.AWS.Region},
		billingService,
	).WithDatabase(db)
	messageTemplateService := messagetemplatemodule.NewService(db, messageTemplateRepository, emailAPIService)
	broadcastService := broadcastmodule.NewService(broadcastRepository, messageTemplateService)
	webhookService := webhooksmodule.NewService(db, webhookRepository, webhookEmitter)
	dnsVerifier := netdns.New()
	domainService := domainmodule.NewService(domainRepository, registry.emailClient, dnsVerifier, emailTenantService).WithDatabase(db).WithNotifier(notificationEmailService)
	domainClaimRepository := domainclaimmodule.NewRepository(db)
	domainClaimService := domainclaimmodule.NewService(db, domainClaimRepository, registry.emailClient, dnsVerifier, emailTenantService)
	walletService := walletmodule.NewService(
		walletmodule.NewRepository(db),
		walletmodule.ServiceConfig{FrontendURL: cfg.FrontendURL, BackendURL: cfg.BackendURL},
		registry.hubtelProvider,
		registry.hubtelPayments,
	)

	middleware := newServerMiddleware(cfg.IsDevelopment(), cfg.CORSOrigins, sessionRepository, authRepository, teamRepository, teamTokenRepository)

	authmodule.RegisterRoutes(router, authmodule.NewHandler(authService, cfg.IsDevelopment(), cfg.CookieDomain), middleware.auth, middleware.csrf)
	mfamodule.RegisterRoutes(router, mfamodule.NewHandler(mfaService), middleware.auth, middleware.csrf)
	usermodule.RegisterRoutes(router, usermodule.NewHandler(usermodule.NewService(userRepository, notificationEmailService)), middleware.auth, middleware.csrf)
	teammodule.RegisterRoutes(router, teammodule.NewHandler(teamService), middleware.auth, middleware.csrf, middleware.tenant)
	walletmodule.RegisterRoutes(router, walletmodule.NewHandler(walletService), middleware.tenantAccess)
	planmodule.RegisterRoutes(router, planmodule.NewHandler(planmodule.NewService(planmodule.NewRepository(db))), middleware.tenantAccess)
	subscriptionmodule.RegisterRoutes(router, subscriptionmodule.NewHandler(subscriptionmodule.NewService(subscriptionmodule.NewRepository(db)).WithNotifier(notificationEmailService)), middleware.tenantAccess)
	auditeventmodule.RegisterRoutes(router, auditeventmodule.NewHandler(auditeventmodule.NewService(auditRepository)), middleware.auth, middleware.csrf, middleware.tenant)
	teamtokenmodule.RegisterRoutes(router, teamtokenmodule.NewHandler(teamtokenmodule.NewService(teamTokenRepository).WithNotifier(notificationEmailService)), middleware.auth, middleware.csrf, middleware.tenant)
	contactmodule.RegisterRoutes(router, contactmodule.NewHandler(contactmodule.NewService(contactRepository)), middleware.tenantAccess)
	contactpropertymodule.RegisterRoutes(router, contactpropertymodule.NewHandler(contactpropertymodule.NewService(contactPropertyRepository)), middleware.tenantAccess)
	segmentmodule.RegisterRoutes(router, segmentmodule.NewHandler(segmentmodule.NewService(segmentRepository)), middleware.tenantAccess)
	topicmodule.RegisterRoutes(router, topicmodule.NewHandler(topicmodule.NewService(topicRepository)), middleware.tenantAccess)
	suppressionmodule.RegisterRoutes(router, suppressionmodule.NewHandler(suppressionmodule.NewService(suppressionRepository)), middleware.tenantAccess)
	messagetemplatemodule.RegisterRoutes(router, messagetemplatemodule.NewHandler(messageTemplateService), middleware.tenantAccess)
	broadcastmodule.RegisterRoutes(router, broadcastmodule.NewHandler(broadcastService), middleware.tenantAccess)
	senderidmodule.RegisterRoutes(router, senderidmodule.NewHandler(senderidmodule.NewService(senderIDRepository)), middleware.tenantAccess)
	domainmodule.RegisterRoutes(router, domainmodule.NewHandler(domainService), middleware.tenantAccess)
	domainclaimmodule.RegisterRoutes(router, domainclaimmodule.NewHandler(domainClaimService), middleware.tenantAccess)
	smsmodule.RegisterRoutes(router, smsmodule.NewHandler(smsService), middleware.tenantAccess)
	smscampaignmodule.RegisterRoutes(router, smscampaignmodule.NewHandler(smscampaignmodule.NewService(smsCampaignRepository)), middleware.tenantAccess)
	emailmodule.RegisterRoutes(router, emailmodule.NewHandler(emailAPIService), middleware.tenantAccess)
	webhooksmodule.RegisterRoutes(router, webhooksmodule.NewHandler(webhookService), middleware.auth, middleware.csrf, middleware.tenant)
	sessionmodule.RegisterRoutes(router, sessionmodule.NewHandler(sessionmodule.NewService(sessionRepository)), middleware.auth, middleware.csrf)
	return nil
}

func newServerMiddleware(
	development bool,
	corsOrigins []string,
	sessionRepository *sessionmodule.Repository,
	authRepository *authmodule.Repository,
	teamRepository *teammodule.Repository,
	teamTokenRepository *teamtokenmodule.Repository,
) serverMiddleware {
	authMiddleware := httpmiddleware.SessionAuth(httpmiddleware.SessionAuthConfig{Sessions: sessionRepository, Users: authRepository})
	csrfMiddleware := httpmiddleware.CSRF(httpmiddleware.CSRFConfig{Development: development, TrustedOrigins: corsOrigins})
	resolver := httpmiddleware.CredentialResolver{Sessions: sessionRepository, Users: authRepository, Tokens: teamTokenRepository}
	authenticate := httpmiddleware.Authenticate(httpmiddleware.AuthenticateConfig{Resolver: resolver, CSRF: csrfMiddleware})
	selectTeam := httpmiddleware.SelectTeam(httpmiddleware.SelectTeamConfig{Memberships: teamRepository})
	tenantMiddleware := func(permission authz.Permission) echo.MiddlewareFunc {
		return httpmiddleware.Tenant(httpmiddleware.TenantConfig{Memberships: teamRepository, Required: permission})
	}
	tenantAccess := func(permission authz.Permission) echo.MiddlewareFunc {
		return httpmiddleware.Chain(authenticate, selectTeam, httpmiddleware.Authorize(permission))
	}
	return serverMiddleware{auth: authMiddleware, csrf: csrfMiddleware, tenant: tenantMiddleware, tenantAccess: tenantAccess}
}

func defaultHTTPMiddleware() []echo.MiddlewareFunc {
	return []echo.MiddlewareFunc{
		httpmiddleware.NewRelic(),
		sentryecho.New(sentryecho.Options{Repanic: true, WaitForDelivery: false}),
		httpmiddleware.SentryErrors(),
	}
}
