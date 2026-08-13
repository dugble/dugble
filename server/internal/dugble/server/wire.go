package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/coffeyvidzro/dugble/server/internal/adapters/dns/netdns"
	newrelicmonitoring "github.com/coffeyvidzro/dugble/server/internal/adapters/monitoring/newrelic"
	sentrymonitoring "github.com/coffeyvidzro/dugble/server/internal/adapters/monitoring/sentry"
	"github.com/coffeyvidzro/dugble/server/internal/adapters/postgres"
	redisadapter "github.com/coffeyvidzro/dugble/server/internal/adapters/redis"
	platformbilling "github.com/coffeyvidzro/dugble/server/internal/billing/charge/usage"
	planmodule "github.com/coffeyvidzro/dugble/server/internal/billing/plan"
	subscriptionmodule "github.com/coffeyvidzro/dugble/server/internal/billing/subscription"
	walletmodule "github.com/coffeyvidzro/dugble/server/internal/billing/wallet"
	"github.com/coffeyvidzro/dugble/server/internal/config"
	emaildelivery "github.com/coffeyvidzro/dugble/server/internal/delivery/email/outbound"
	smsdelivery "github.com/coffeyvidzro/dugble/server/internal/delivery/sms/outbound"
	auditeventmodule "github.com/coffeyvidzro/dugble/server/internal/modules/auditevent"
	authmodule "github.com/coffeyvidzro/dugble/server/internal/modules/auth"
	broadcastmodule "github.com/coffeyvidzro/dugble/server/internal/modules/broadcast"
	smscampaignmodule "github.com/coffeyvidzro/dugble/server/internal/modules/campaign"
	contactmodule "github.com/coffeyvidzro/dugble/server/internal/modules/contact"
	contactpropertymodule "github.com/coffeyvidzro/dugble/server/internal/modules/contactproperty"
	domainmodule "github.com/coffeyvidzro/dugble/server/internal/modules/domain"
	emailmodule "github.com/coffeyvidzro/dugble/server/internal/modules/email"
	"github.com/coffeyvidzro/dugble/server/internal/modules/emailtenant"
	tenantprovision "github.com/coffeyvidzro/dugble/server/internal/modules/emailtenant/provisioning"
	healthmodule "github.com/coffeyvidzro/dugble/server/internal/modules/health"
	messagetemplatemodule "github.com/coffeyvidzro/dugble/server/internal/modules/messagetemplate"
	mfamodule "github.com/coffeyvidzro/dugble/server/internal/modules/mfa"
	segmentmodule "github.com/coffeyvidzro/dugble/server/internal/modules/segment"
	senderidmodule "github.com/coffeyvidzro/dugble/server/internal/modules/senderid"
	sessionmodule "github.com/coffeyvidzro/dugble/server/internal/modules/session"
	smsmodule "github.com/coffeyvidzro/dugble/server/internal/modules/sms"
	suppressionmodule "github.com/coffeyvidzro/dugble/server/internal/modules/suppression"
	teammodule "github.com/coffeyvidzro/dugble/server/internal/modules/team"
	teamtokenmodule "github.com/coffeyvidzro/dugble/server/internal/modules/teamtoken"
	topicmodule "github.com/coffeyvidzro/dugble/server/internal/modules/topic"
	usermodule "github.com/coffeyvidzro/dugble/server/internal/modules/user"
	webhooksmodule "github.com/coffeyvidzro/dugble/server/internal/modules/webhooks"
	"github.com/coffeyvidzro/dugble/server/internal/platform/audit"
	"github.com/coffeyvidzro/dugble/server/internal/platform/outbox"
	"github.com/coffeyvidzro/dugble/server/internal/platform/systemmail"
	platformwebhook "github.com/coffeyvidzro/dugble/server/internal/platform/webhook"
	"github.com/coffeyvidzro/dugble/server/internal/security"
	httptransport "github.com/coffeyvidzro/dugble/server/internal/transport"
)

// Wire builds the server and returns a cleanup function for all initialized resources.
func Wire(ctx context.Context) (*Application, func(), error) {
	if ctx == nil {
		return nil, nil, errors.New("server wiring context is required")
	}

	cleanups := &cleanupStack{}
	fail := func(err error) (*Application, func(), error) {
		cleanups.Run()
		return nil, nil, err
	}

	cfg, err := config.Load()
	if err != nil {
		return fail(fmt.Errorf("load configuration: %w", err))
	}
	if err := sentrymonitoring.Init(cfg.Sentry, cfg.AppEnv); err != nil {
		return fail(fmt.Errorf("initialize Sentry: %w", err))
	}
	cleanups.Add(func() { sentrymonitoring.Flush(5 * time.Second) })

	newRelic, err := newrelicmonitoring.New("dugble-api", cfg.AppEnv, cfg.NewRelic)
	if err != nil {
		return fail(fmt.Errorf("initialize New Relic client: %w", err))
	}
	cleanups.Add(func() { newrelicmonitoring.Shutdown(newRelic, 5*time.Second) })

	startupCtx, cancelStartup := context.WithTimeout(ctx, 15*time.Second)
	defer cancelStartup()

	db, err := postgres.New(startupCtx, cfg.DatabaseURL)
	if err != nil {
		return fail(fmt.Errorf("initialize PostgreSQL: %w", err))
	}
	cleanups.Add(db.Close)

	redisClient, err := redisadapter.New(startupCtx, cfg.RedisURL)
	if err != nil {
		return fail(fmt.Errorf("initialize Redis: %w", err))
	}
	cleanups.Add(func() {
		if closeErr := redisClient.Close(); closeErr != nil {
			sentrymonitoring.Warn("close Redis client", "error", closeErr)
		}
	})

	arcjetClient, err := newArcjetClient(cfg)
	if err != nil {
		return fail(fmt.Errorf("initialize Arcjet client: %w", err))
	}
	renderer, err := systemmail.NewRenderer()
	if err != nil {
		return fail(fmt.Errorf("initialize system email renderer: %w", err))
	}
	emailClient, err := newEmailClient(cfg)
	if err != nil {
		return fail(fmt.Errorf("initialize email client: %w", err))
	}

	outboxRepository := outbox.NewRepository(db)
	systemEmailQueue := newSystemEmailQueue(cfg, outboxRepository)
	snsHandler := newProviderSNSHandler(cfg, db, outboxRepository)
	hubtelProvider, hubtelPayments := newHubtelServices(cfg, db)
	smsSender, err := newSMSSender(cfg)
	if err != nil {
		return fail(err)
	}

	notificationEmailService := systemmail.NewEmailService(
		systemEmailQueue,
		renderer,
		cfg.FrontendURL,
		cfg.AWS.FromEmail,
	)
	auditRepository := audit.NewRepository(db)
	audit.SetSink(auditRepository)
	sessionRepository := sessionmodule.NewRepository(db)
	authRepository := authmodule.NewRepository(db)
	mfaCipher, err := security.NewSecretCipherKeyring(cfg.EncryptionKeys)
	if err != nil {
		return fail(fmt.Errorf("initialize MFA cipher: %w", err))
	}
	mfaService := mfamodule.NewService(
		mfamodule.NewRepository(db),
		mfaCipher,
		"Dugble",
	).WithNotifier(notificationEmailService)
	authService := authmodule.NewService(
		authRepository,
		sessionRepository,
		notificationEmailService,
		mfaService,
	)
	userRepository := usermodule.NewRepository(db)
	mfaService.WithRecipientStore(userRepository)
	teamRepository := teammodule.NewRepository(db)
	teamService := teammodule.NewService(
		teamRepository,
		notificationEmailService,
	).WithRecipientStore(userRepository)
	teamTokenRepository := teamtokenmodule.NewRepository(db)
	contactRepository := contactmodule.NewRepository(db)
	contactPropertyRepository := contactpropertymodule.NewRepository(db)
	segmentRepository := segmentmodule.NewRepository(db)
	topicRepository := topicmodule.NewRepository(db)
	suppressionRepository := suppressionmodule.NewRepository(db)
	messageTemplateRepository := messagetemplatemodule.NewRepository(db)
	broadcastRepository := broadcastmodule.NewRepository(db)
	domainRepository := domainmodule.NewRepository(db)
	emailTenantRepository := emailtenant.NewRepository(db)
	emailTenantService := emailtenant.NewService(
		emailTenantRepository,
		tenantprovision.NewQueue(outboxRepository),
	)
	senderIDRepository := senderidmodule.NewRepository(db)
	webhookRepository := webhooksmodule.NewRepository(db)
	webhookEmitter := platformwebhook.NewEmitter(webhookRepository)
	smsRepository := smsmodule.NewRepositoryWithWebhookEmitter(db, webhookEmitter)
	smsCampaignRepository := smscampaignmodule.NewRepository(db)
	billingService := platformbilling.NewService(platformbilling.NewRepository(db))
	smsService := smsmodule.NewService(
		smsRepository,
		smsSender,
		smsdelivery.NewQueue(outboxRepository),
		billingService,
	)
	emailRepository := emailmodule.NewRepository(db)
	emailAPIService := emailmodule.NewService(
		emailRepository,
		emaildelivery.NewQueue(outboxRepository),
		emailmodule.ServiceConfig{
			DefaultFromEmail: cfg.AWS.FromEmail,
			DefaultProvider:  domainmodule.DefaultProvider,
			DefaultRegion:    cfg.AWS.Region,
		},
		billingService,
	)
	messageTemplateService := messagetemplatemodule.NewService(messageTemplateRepository, emailAPIService)
	broadcastService := broadcastmodule.NewService(broadcastRepository, messageTemplateService)
	webhookService := webhooksmodule.NewService(webhookRepository, webhookEmitter)
	domainService := domainmodule.NewService(domainRepository, emailClient, netdns.New(), emailTenantService)
	walletService := walletmodule.NewService(
		walletmodule.NewRepository(db),
		walletmodule.ServiceConfig{FrontendURL: cfg.FrontendURL, BackendURL: cfg.BackendURL},
		hubtelProvider,
		hubtelPayments,
	)

	serverMiddleware := newServerMiddleware(serverMiddlewareDependencies{
		config:              cfg,
		sessionRepository:   sessionRepository,
		authRepository:      authRepository,
		teamRepository:      teamRepository,
		teamTokenRepository: teamTokenRepository,
	})
	routeHandlers := serverRouteHandlers{
		health:          healthmodule.NewHandler(db, redisClient),
		providerSNS:     snsHandler,
		auth:            authmodule.NewHandler(authService, cfg.IsDevelopment(), cfg.CookieDomain),
		mfa:             mfamodule.NewHandler(mfaService),
		user:            usermodule.NewHandler(usermodule.NewService(userRepository, notificationEmailService)),
		team:            teammodule.NewHandler(teamService),
		wallet:          walletmodule.NewHandler(walletService),
		plan:            planmodule.NewHandler(planmodule.NewService(planmodule.NewRepository(db))),
		subscription:    subscriptionmodule.NewHandler(subscriptionmodule.NewService(subscriptionmodule.NewRepository(db))),
		auditEvent:      auditeventmodule.NewHandler(auditeventmodule.NewService(auditRepository)),
		contact:         contactmodule.NewHandler(contactmodule.NewService(contactRepository)),
		contactProperty: contactpropertymodule.NewHandler(contactpropertymodule.NewService(contactPropertyRepository)),
		segment:         segmentmodule.NewHandler(segmentmodule.NewService(segmentRepository)),
		topic:           topicmodule.NewHandler(topicmodule.NewService(topicRepository)),
		suppression:     suppressionmodule.NewHandler(suppressionmodule.NewService(suppressionRepository)),
		messageTemplate: messagetemplatemodule.NewHandler(messageTemplateService),
		broadcast:       broadcastmodule.NewHandler(broadcastService),
		teamToken: teamtokenmodule.NewHandler(
			teamtokenmodule.NewService(teamTokenRepository).WithNotifier(notificationEmailService),
		),
		senderID:    senderidmodule.NewHandler(senderidmodule.NewService(senderIDRepository)),
		domain:      domainmodule.NewHandler(domainService),
		sms:         smsmodule.NewHandler(smsService),
		smsCampaign: smscampaignmodule.NewHandler(smscampaignmodule.NewService(smsCampaignRepository)),
		email:       emailmodule.NewHandler(emailAPIService),
		webhooks:    webhooksmodule.NewHandler(webhookService),
		session:     sessionmodule.NewHandler(sessionmodule.NewService(sessionRepository)),
	}

	router, err := httptransport.NewRouter(
		newRouterConfig(cfg, arcjetClient, db),
		newRouteRegistrar(routeHandlers, serverMiddleware),
	)
	if err != nil {
		return fail(fmt.Errorf("create HTTP router: %w", err))
	}

	application, err := NewApplication(newrelicmonitoring.WrapHTTP(newRelic, router), ":"+cfg.HTTPPort)
	if err != nil {
		return fail(fmt.Errorf("create HTTP application: %w", err))
	}
	return application, cleanups.Run, nil
}

type cleanupStack struct {
	functions []func()
}

func (stack *cleanupStack) Add(cleanup func()) {
	if stack == nil || cleanup == nil {
		return
	}
	stack.functions = append(stack.functions, cleanup)
}

func (stack *cleanupStack) Run() {
	if stack == nil {
		return
	}
	for index := len(stack.functions) - 1; index >= 0; index-- {
		stack.functions[index]()
	}
	stack.functions = nil
}
