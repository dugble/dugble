package backoffice

import (
	"errors"

	"github.com/labstack/echo/v5"

	backofficeallowances "github.com/dugble/dugble/server/internal/backoffice/allowances"
	backofficeauditlog "github.com/dugble/dugble/server/internal/backoffice/auditlog"
	backofficecurrencies "github.com/dugble/dugble/server/internal/backoffice/currencies"
	backofficedashboard "github.com/dugble/dugble/server/internal/backoffice/dashboard"
	backofficedomains "github.com/dugble/dugble/server/internal/backoffice/domains"
	backofficeemail "github.com/dugble/dugble/server/internal/backoffice/email"
	backofficefxrates "github.com/dugble/dugble/server/internal/backoffice/fxrates"
	backofficehealth "github.com/dugble/dugble/server/internal/backoffice/health"
	backofficemarkets "github.com/dugble/dugble/server/internal/backoffice/markets"
	backofficepayments "github.com/dugble/dugble/server/internal/backoffice/payments"
	backofficeproductrates "github.com/dugble/dugble/server/internal/backoffice/productrates"
	backofficesenderids "github.com/dugble/dugble/server/internal/backoffice/senderids"
	backofficesms "github.com/dugble/dugble/server/internal/backoffice/sms"
	backofficesmsrates "github.com/dugble/dugble/server/internal/backoffice/smsrates"
	backofficesubscriptions "github.com/dugble/dugble/server/internal/backoffice/subscriptions"
	backofficeteams "github.com/dugble/dugble/server/internal/backoffice/teams"
	backofficeusers "github.com/dugble/dugble/server/internal/backoffice/users"
	backofficewallets "github.com/dugble/dugble/server/internal/backoffice/wallets"
	httptransport "github.com/dugble/dugble/server/internal/transport"
)

type routeHandlers struct {
	health        *backofficehealth.Handler
	dashboard     *backofficedashboard.Handler
	auditLog      *backofficeauditlog.Handler
	users         *backofficeusers.Handler
	teams         *backofficeteams.Handler
	sms           *backofficesms.Handler
	senderIDs     *backofficesenderids.Handler
	domains       *backofficedomains.Handler
	email         *backofficeemail.Handler
	currencies    *backofficecurrencies.Handler
	markets       *backofficemarkets.Handler
	smsRates      *backofficesmsrates.Handler
	productRates  *backofficeproductrates.Handler
	fxRates       *backofficefxrates.Handler
	allowances    *backofficeallowances.Handler
	wallets       *backofficewallets.Handler
	payments      *backofficepayments.Handler
	subscriptions *backofficesubscriptions.Handler
}

func newRouteRegistrar(
	handlers routeHandlers,
	backofficeAccess ...echo.MiddlewareFunc,
) httptransport.Registrar {
	return func(router *echo.Echo) error {
		if router == nil {
			return errors.New("backoffice router is required")
		}
		if handlers.health == nil {
			return errors.New("backoffice health handler is required")
		}
		backofficehealth.RegisterRoutes(router, handlers.health)

		if len(backofficeAccess) == 0 {
			return nil
		}
		if handlers.dashboard == nil ||
			handlers.auditLog == nil ||
			handlers.users == nil ||
			handlers.teams == nil ||
			handlers.sms == nil ||
			handlers.senderIDs == nil ||
			handlers.domains == nil ||
			handlers.email == nil ||
			handlers.currencies == nil ||
			handlers.markets == nil ||
			handlers.smsRates == nil ||
			handlers.productRates == nil ||
			handlers.fxRates == nil ||
			handlers.allowances == nil ||
			handlers.wallets == nil || handlers.payments == nil || handlers.subscriptions == nil {
			return errors.New("backoffice administrative handlers are required")
		}

		backofficedashboard.RegisterRoutes(router, handlers.dashboard, backofficeAccess...)
		backofficeauditlog.RegisterRoutes(router, handlers.auditLog, backofficeAccess...)
		backofficeusers.RegisterRoutes(router, handlers.users, backofficeAccess...)
		backofficeteams.RegisterRoutes(router, handlers.teams, backofficeAccess...)
		backofficesms.RegisterRoutes(router, handlers.sms, backofficeAccess...)
		backofficesenderids.RegisterRoutes(router, handlers.senderIDs, backofficeAccess...)
		backofficedomains.RegisterRoutes(router, handlers.domains, backofficeAccess...)
		backofficeemail.RegisterRoutes(router, handlers.email, backofficeAccess...)
		backofficecurrencies.RegisterRoutes(router, handlers.currencies, backofficeAccess...)
		backofficemarkets.RegisterRoutes(router, handlers.markets, backofficeAccess...)
		backofficesmsrates.RegisterRoutes(router, handlers.smsRates, backofficeAccess...)
		backofficeproductrates.RegisterRoutes(router, handlers.productRates, backofficeAccess...)
		backofficefxrates.RegisterRoutes(router, handlers.fxRates, backofficeAccess...)
		backofficeallowances.RegisterRoutes(router, handlers.allowances, backofficeAccess...)
		backofficewallets.RegisterRoutes(router, handlers.wallets, backofficeAccess...)
		backofficepayments.RegisterRoutes(router, handlers.payments, backofficeAccess...)
		backofficesubscriptions.RegisterRoutes(router, handlers.subscriptions, backofficeAccess...)
		return nil
	}
}
