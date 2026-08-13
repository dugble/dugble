package server

import (
	"github.com/labstack/echo/v5"

	planhttp "github.com/coffeyvidzro/dugble/server/internal/billing/plan"
	subscriptionhttp "github.com/coffeyvidzro/dugble/server/internal/billing/subscription"
	wallethttp "github.com/coffeyvidzro/dugble/server/internal/billing/wallet"
	auditeventhttp "github.com/coffeyvidzro/dugble/server/internal/modules/auditevent"
	authhttp "github.com/coffeyvidzro/dugble/server/internal/modules/auth"
	broadcasthttp "github.com/coffeyvidzro/dugble/server/internal/modules/broadcast"
	smscampaignhttp "github.com/coffeyvidzro/dugble/server/internal/modules/campaign"
	contacthttp "github.com/coffeyvidzro/dugble/server/internal/modules/contact"
	contactpropertyhttp "github.com/coffeyvidzro/dugble/server/internal/modules/contactproperty"
	domainhttp "github.com/coffeyvidzro/dugble/server/internal/modules/domain"
	emailhttp "github.com/coffeyvidzro/dugble/server/internal/modules/email"
	healthhttp "github.com/coffeyvidzro/dugble/server/internal/modules/health"
	messagetemplatehttp "github.com/coffeyvidzro/dugble/server/internal/modules/messagetemplate"
	mfahttp "github.com/coffeyvidzro/dugble/server/internal/modules/mfa"
	segmenthttp "github.com/coffeyvidzro/dugble/server/internal/modules/segment"
	senderidhttp "github.com/coffeyvidzro/dugble/server/internal/modules/senderid"
	sessionhttp "github.com/coffeyvidzro/dugble/server/internal/modules/session"
	smshttp "github.com/coffeyvidzro/dugble/server/internal/modules/sms"
	suppressionhttp "github.com/coffeyvidzro/dugble/server/internal/modules/suppression"
	teamhttp "github.com/coffeyvidzro/dugble/server/internal/modules/team"
	teamtokenhttp "github.com/coffeyvidzro/dugble/server/internal/modules/teamtoken"
	topichttp "github.com/coffeyvidzro/dugble/server/internal/modules/topic"
	userhttp "github.com/coffeyvidzro/dugble/server/internal/modules/user"
	webhookshttp "github.com/coffeyvidzro/dugble/server/internal/modules/webhooks"
	httptransport "github.com/coffeyvidzro/dugble/server/internal/transport"
	httpmiddleware "github.com/coffeyvidzro/dugble/server/internal/transport/middleware"
	providersns "github.com/coffeyvidzro/dugble/server/internal/transport/provider/aws/sns"
	apperrors "github.com/coffeyvidzro/dugble/server/pkg/errors"
	"github.com/coffeyvidzro/dugble/server/pkg/httputil"
)

type serverRouteHandlers struct {
	health          *healthhttp.Handler
	providerSNS     *providersns.Handler
	auth            *authhttp.Handler
	mfa             *mfahttp.Handler
	user            *userhttp.Handler
	team            *teamhttp.Handler
	wallet          *wallethttp.Handler
	plan            *planhttp.Handler
	subscription    *subscriptionhttp.Handler
	auditEvent      *auditeventhttp.Handler
	teamToken       *teamtokenhttp.Handler
	contact         *contacthttp.Handler
	contactProperty *contactpropertyhttp.Handler
	segment         *segmenthttp.Handler
	topic           *topichttp.Handler
	suppression     *suppressionhttp.Handler
	messageTemplate *messagetemplatehttp.Handler
	broadcast       *broadcasthttp.Handler
	senderID        *senderidhttp.Handler
	domain          *domainhttp.Handler
	sms             *smshttp.Handler
	smsCampaign     *smscampaignhttp.Handler
	email           *emailhttp.Handler
	webhooks        *webhookshttp.Handler
	session         *sessionhttp.Handler
}

func newRouteRegistrar(handlers serverRouteHandlers, middleware serverMiddleware) httptransport.Registrar {
	return func(router *echo.Echo) error {
		healthhttp.RegisterRoutes(router, handlers.health)
		if handlers.providerSNS != nil {
			providersns.RegisterRoutes(router, handlers.providerSNS)
		}
		registerCSRFRoute(router, middleware.csrf)

		authhttp.RegisterRoutes(router, handlers.auth, middleware.auth, middleware.csrf)
		mfahttp.RegisterRoutes(router, handlers.mfa, middleware.auth, middleware.csrf)
		userhttp.RegisterRoutes(router, handlers.user, middleware.auth, middleware.csrf)
		teamhttp.RegisterRoutes(router, handlers.team, middleware.auth, middleware.csrf, middleware.tenant)
		wallethttp.RegisterRoutes(router, handlers.wallet, middleware.tenantAccess)
		planhttp.RegisterRoutes(router, handlers.plan, middleware.tenantAccess)
		subscriptionhttp.RegisterRoutes(router, handlers.subscription, middleware.tenantAccess)
		auditeventhttp.RegisterRoutes(router, handlers.auditEvent, middleware.auth, middleware.csrf, middleware.tenant)
		teamtokenhttp.RegisterRoutes(router, handlers.teamToken, middleware.auth, middleware.csrf, middleware.tenant)
		contacthttp.RegisterRoutes(router, handlers.contact, middleware.tenantAccess)
		contactpropertyhttp.RegisterRoutes(router, handlers.contactProperty, middleware.tenantAccess)
		segmenthttp.RegisterRoutes(router, handlers.segment, middleware.tenantAccess)
		topichttp.RegisterRoutes(router, handlers.topic, middleware.tenantAccess)
		suppressionhttp.RegisterRoutes(router, handlers.suppression, middleware.tenantAccess)
		messagetemplatehttp.RegisterRoutes(router, handlers.messageTemplate, middleware.tenantAccess)
		broadcasthttp.RegisterRoutes(router, handlers.broadcast, middleware.tenantAccess)
		senderidhttp.RegisterRoutes(router, handlers.senderID, middleware.tenantAccess)
		domainhttp.RegisterRoutes(router, handlers.domain, middleware.tenantAccess)
		smshttp.RegisterRoutes(router, handlers.sms, middleware.tenantAccess)
		smscampaignhttp.RegisterRoutes(router, handlers.smsCampaign, middleware.tenantAccess)
		emailhttp.RegisterRoutes(router, handlers.email, middleware.tenantAccess)
		webhookshttp.RegisterRoutes(router, handlers.webhooks, middleware.auth, middleware.csrf, middleware.tenant)
		sessionhttp.RegisterRoutes(router, handlers.session, middleware.auth, middleware.csrf)
		return nil
	}
}

func registerCSRFRoute(router *echo.Echo, csrfMiddleware echo.MiddlewareFunc) {
	router.GET("/csrf", func(c *echo.Context) error {
		token, ok := c.Get(httpmiddleware.CSRFContextKey).(string)
		if !ok || token == "" {
			return httputil.Error(c, apperrors.NewInternal("CSRF token is not available", nil))
		}
		return httputil.OK(c, map[string]string{"csrf_token": token})
	}, csrfMiddleware)
}
