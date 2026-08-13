package backoffice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/labstack/echo/v5"

	newrelicmonitoring "github.com/coffeyvidzro/dugble/server/internal/adapters/monitoring/newrelic"
	sentrymonitoring "github.com/coffeyvidzro/dugble/server/internal/adapters/monitoring/sentry"
	"github.com/coffeyvidzro/dugble/server/internal/adapters/postgres"
	backofficeallowances "github.com/coffeyvidzro/dugble/server/internal/backoffice/allowances"
	backofficeauditlog "github.com/coffeyvidzro/dugble/server/internal/backoffice/auditlog"
	backofficecurrencies "github.com/coffeyvidzro/dugble/server/internal/backoffice/currencies"
	backofficedashboard "github.com/coffeyvidzro/dugble/server/internal/backoffice/dashboard"
	backofficedomains "github.com/coffeyvidzro/dugble/server/internal/backoffice/domains"
	backofficeemail "github.com/coffeyvidzro/dugble/server/internal/backoffice/email"
	backofficefxrates "github.com/coffeyvidzro/dugble/server/internal/backoffice/fxrates"
	backofficehealth "github.com/coffeyvidzro/dugble/server/internal/backoffice/health"
	backofficemarkets "github.com/coffeyvidzro/dugble/server/internal/backoffice/markets"
	backofficepayments "github.com/coffeyvidzro/dugble/server/internal/backoffice/payments"
	backofficeproductrates "github.com/coffeyvidzro/dugble/server/internal/backoffice/productrates"
	backofficesenderids "github.com/coffeyvidzro/dugble/server/internal/backoffice/senderids"
	backofficesms "github.com/coffeyvidzro/dugble/server/internal/backoffice/sms"
	backofficesmsrates "github.com/coffeyvidzro/dugble/server/internal/backoffice/smsrates"
	backofficesubscriptions "github.com/coffeyvidzro/dugble/server/internal/backoffice/subscriptions"
	backofficeteams "github.com/coffeyvidzro/dugble/server/internal/backoffice/teams"
	backofficeusers "github.com/coffeyvidzro/dugble/server/internal/backoffice/users"
	backofficewallets "github.com/coffeyvidzro/dugble/server/internal/backoffice/wallets"
	"github.com/coffeyvidzro/dugble/server/internal/config"
	authmodule "github.com/coffeyvidzro/dugble/server/internal/modules/auth"
	sessionmodule "github.com/coffeyvidzro/dugble/server/internal/modules/session"
	httptransport "github.com/coffeyvidzro/dugble/server/internal/transport"
	httpmiddleware "github.com/coffeyvidzro/dugble/server/internal/transport/middleware"
)

// Wire builds the backoffice process and returns cleanup for all resources.
func Wire(ctx context.Context) (*Application, func(), error) {
	if ctx == nil {
		return nil, nil, errors.New("backoffice wiring context is required")
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

	newRelic, err := newrelicmonitoring.New(serviceName, cfg.AppEnv, cfg.NewRelic)
	if err != nil {
		return fail(fmt.Errorf("initialize New Relic: %w", err))
	}
	cleanups.Add(func() { newrelicmonitoring.Shutdown(newRelic, 5*time.Second) })

	startupCtx, cancelStartup := context.WithTimeout(ctx, 15*time.Second)
	defer cancelStartup()

	db, err := postgres.New(startupCtx, cfg.DatabaseURL)
	if err != nil {
		return fail(fmt.Errorf("initialize PostgreSQL: %w", err))
	}
	cleanups.Add(db.Close)

	sessionRepository := sessionmodule.NewRepository(db)
	authRepository := authmodule.NewRepository(db)
	authMiddleware := httpmiddleware.SessionAuth(httpmiddleware.SessionAuthConfig{
		Sessions: sessionRepository,
		Users:    authRepository,
	})
	adminMiddleware := RequireAdmin(cfg.Backoffice.AdminEmails)
	csrfMiddleware := httpmiddleware.CSRF(httpmiddleware.CSRFConfig{
		Development:    cfg.IsDevelopment(),
		TrustedOrigins: cfg.CORSOrigins,
		TokenLookup:    "header:" + echo.HeaderXCSRFToken,
		CookieName:     "dugble_backoffice_csrf",
	})

	dashboardService := backofficedashboard.NewService(backofficedashboard.NewRepository(db))
	auditLogService := backofficeauditlog.NewService(backofficeauditlog.NewRepository(db))
	usersService := backofficeusers.NewService(backofficeusers.NewRepository(db))
	teamsService := backofficeteams.NewService(backofficeteams.NewRepository(db))
	smsService := backofficesms.NewService(backofficesms.NewRepository(db))
	senderIDsService := backofficesenderids.NewService(backofficesenderids.NewRepository(db))
	domainsService := backofficedomains.NewService(backofficedomains.NewRepository(db))
	emailService := backofficeemail.NewService(backofficeemail.NewRepository(db))
	currenciesService := backofficecurrencies.NewService(backofficecurrencies.NewRepository(db))
	marketsService := backofficemarkets.NewService(backofficemarkets.NewRepository(db))
	smsRatesService := backofficesmsrates.NewService(backofficesmsrates.NewRepository(db))
	productRatesService := backofficeproductrates.NewService(backofficeproductrates.NewRepository(db))
	fxRatesService := backofficefxrates.NewService(backofficefxrates.NewRepository(db))
	allowancesService := backofficeallowances.NewService(backofficeallowances.NewRepository(db))
	walletsService := backofficewallets.NewService(backofficewallets.NewRepository(db))
	paymentsService := backofficepayments.NewService(backofficepayments.NewRepository(db))
	subscriptionsService := backofficesubscriptions.NewService(backofficesubscriptions.NewRepository(db))

	handlers := routeHandlers{
		health:        backofficehealth.NewHandler(db),
		dashboard:     backofficedashboard.NewHandler(dashboardService),
		auditLog:      backofficeauditlog.NewHandler(auditLogService),
		users:         backofficeusers.NewHandler(usersService),
		teams:         backofficeteams.NewHandler(teamsService),
		sms:           backofficesms.NewHandler(smsService),
		senderIDs:     backofficesenderids.NewHandler(senderIDsService),
		domains:       backofficedomains.NewHandler(domainsService),
		email:         backofficeemail.NewHandler(emailService),
		currencies:    backofficecurrencies.NewHandler(currenciesService),
		markets:       backofficemarkets.NewHandler(marketsService),
		smsRates:      backofficesmsrates.NewHandler(smsRatesService),
		productRates:  backofficeproductrates.NewHandler(productRatesService),
		fxRates:       backofficefxrates.NewHandler(fxRatesService),
		allowances:    backofficeallowances.NewHandler(allowancesService),
		wallets:       backofficewallets.NewHandler(walletsService),
		payments:      backofficepayments.NewHandler(paymentsService),
		subscriptions: backofficesubscriptions.NewHandler(subscriptionsService),
	}
	router, err := httptransport.NewRouter(
		httptransport.RouterConfig{
			Development: cfg.IsDevelopment(),
			CORSOrigins: cfg.CORSOrigins,
		},
		newRouteRegistrar(handlers, authMiddleware, adminMiddleware, csrfMiddleware),
	)
	if err != nil {
		return fail(fmt.Errorf("create backoffice HTTP router: %w", err))
	}

	application, err := NewApplication(
		newrelicmonitoring.WrapHTTP(newRelic, router),
		":"+cfg.Backoffice.HTTPPort,
	)
	if err != nil {
		return fail(fmt.Errorf("create backoffice HTTP application: %w", err))
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
