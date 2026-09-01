package backoffice

import (
	"github.com/labstack/echo/v5"

	backofficeallowances "github.com/dugble/dugble/server/internal/backoffice/allowances"
	backofficeauditlog "github.com/dugble/dugble/server/internal/backoffice/auditlog"
	backofficecurrencies "github.com/dugble/dugble/server/internal/backoffice/currencies"
	backofficedashboard "github.com/dugble/dugble/server/internal/backoffice/dashboard"
	backofficedomains "github.com/dugble/dugble/server/internal/backoffice/domains"
	backofficeemail "github.com/dugble/dugble/server/internal/backoffice/email"
	backofficefxrates "github.com/dugble/dugble/server/internal/backoffice/fxrates"
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
)

func (registry *Registry) registerModules(router *echo.Echo, access ...echo.MiddlewareFunc) {
	db := registry.postgres

	backofficedashboard.RegisterRoutes(
		router,
		backofficedashboard.NewHandler(backofficedashboard.NewService(backofficedashboard.NewRepository(db))),
		access...,
	)
	backofficeauditlog.RegisterRoutes(
		router,
		backofficeauditlog.NewHandler(backofficeauditlog.NewService(backofficeauditlog.NewRepository(db))),
		access...,
	)
	backofficeusers.RegisterRoutes(
		router,
		backofficeusers.NewHandler(backofficeusers.NewService(backofficeusers.NewRepository(db))),
		access...,
	)
	backofficeteams.RegisterRoutes(
		router,
		backofficeteams.NewHandler(backofficeteams.NewService(backofficeteams.NewRepository(db))),
		access...,
	)
	backofficesms.RegisterRoutes(
		router,
		backofficesms.NewHandler(backofficesms.NewService(backofficesms.NewRepository(db))),
		access...,
	)
	backofficesenderids.RegisterRoutes(
		router,
		backofficesenderids.NewHandler(backofficesenderids.NewService(backofficesenderids.NewRepository(db))),
		access...,
	)
	backofficedomains.RegisterRoutes(
		router,
		backofficedomains.NewHandler(backofficedomains.NewService(backofficedomains.NewRepository(db))),
		access...,
	)
	backofficeemail.RegisterRoutes(
		router,
		backofficeemail.NewHandler(backofficeemail.NewService(backofficeemail.NewRepository(db))),
		access...,
	)
	backofficecurrencies.RegisterRoutes(
		router,
		backofficecurrencies.NewHandler(backofficecurrencies.NewService(backofficecurrencies.NewRepository(db))),
		access...,
	)
	backofficemarkets.RegisterRoutes(
		router,
		backofficemarkets.NewHandler(backofficemarkets.NewService(backofficemarkets.NewRepository(db))),
		access...,
	)
	backofficesmsrates.RegisterRoutes(
		router,
		backofficesmsrates.NewHandler(backofficesmsrates.NewService(backofficesmsrates.NewRepository(db))),
		access...,
	)
	backofficeproductrates.RegisterRoutes(
		router,
		backofficeproductrates.NewHandler(backofficeproductrates.NewService(backofficeproductrates.NewRepository(db))),
		access...,
	)
	backofficefxrates.RegisterRoutes(
		router,
		backofficefxrates.NewHandler(backofficefxrates.NewService(backofficefxrates.NewRepository(db))),
		access...,
	)
	backofficeallowances.RegisterRoutes(
		router,
		backofficeallowances.NewHandler(backofficeallowances.NewService(backofficeallowances.NewRepository(db))),
		access...,
	)
	backofficewallets.RegisterRoutes(
		router,
		backofficewallets.NewHandler(backofficewallets.NewService(backofficewallets.NewRepository(db))),
		access...,
	)
	backofficepayments.RegisterRoutes(
		router,
		backofficepayments.NewHandler(backofficepayments.NewService(backofficepayments.NewRepository(db))),
		access...,
	)
	backofficesubscriptions.RegisterRoutes(
		router,
		backofficesubscriptions.NewHandler(backofficesubscriptions.NewService(backofficesubscriptions.NewRepository(db))),
		access...,
	)
}
