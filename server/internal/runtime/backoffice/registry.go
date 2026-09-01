package backoffice

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	newrelicmonitoring "github.com/dugble/dugble/server/internal/integrations/monitoring/newrelic"
	sentrymonitoring "github.com/dugble/dugble/server/internal/integrations/monitoring/sentry"
	"github.com/dugble/dugble/server/internal/integrations/postgres"
	"github.com/dugble/dugble/server/internal/platform/config"
	httptransport "github.com/dugble/dugble/server/internal/runtime/http"
)

const serviceName = "dugble-backoffice"

// Registry owns the backoffice's long-lived infrastructure and HTTP lifecycle.
type Registry struct {
	config   *config.Config
	postgres *pgxpool.Pool
	server   *httptransport.Server
	cleanups cleanupStack
}

// Start wires and runs backoffice until an interrupt or termination signal.
func Start() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	registry, err := New(ctx)
	if err != nil {
		return err
	}
	defer registry.Close()
	return registry.Run(ctx)
}

// New constructs the backoffice registry and all long-lived infrastructure.
func New(ctx context.Context) (*Registry, error) {
	if ctx == nil {
		return nil, errors.New("backoffice registry context is required")
	}

	registry := &Registry{}
	fail := func(err error) (*Registry, error) {
		registry.Close()
		return nil, err
	}

	cfg, err := config.Load()
	if err != nil {
		return fail(fmt.Errorf("load configuration: %w", err))
	}
	registry.config = cfg

	if err := sentrymonitoring.Init(cfg.Sentry, cfg.AppEnv); err != nil {
		return fail(fmt.Errorf("initialize Sentry: %w", err))
	}
	registry.cleanups.Add(func() { sentrymonitoring.Flush(5 * time.Second) })

	newRelic, err := newrelicmonitoring.New(serviceName, cfg.AppEnv, cfg.NewRelic)
	if err != nil {
		return fail(fmt.Errorf("initialize New Relic: %w", err))
	}
	registry.cleanups.Add(func() { newrelicmonitoring.Shutdown(newRelic, 5*time.Second) })

	startupCtx, cancelStartup := context.WithTimeout(ctx, 15*time.Second)
	defer cancelStartup()

	registry.postgres, err = postgres.New(startupCtx, cfg.DatabaseURL)
	if err != nil {
		return fail(fmt.Errorf("initialize PostgreSQL: %w", err))
	}
	registry.cleanups.Add(registry.postgres.Close)

	router, err := httptransport.NewRouter(
		httptransport.RouterConfig{
			Development: cfg.IsDevelopment(),
			CORSOrigins: cfg.CORSOrigins,
		},
		registry.registerRoutes,
	)
	if err != nil {
		return fail(fmt.Errorf("create backoffice HTTP router: %w", err))
	}

	registry.server, err = httptransport.NewServer(
		newrelicmonitoring.WrapHTTP(newRelic, router),
		":"+cfg.Backoffice.HTTPPort,
	)
	if err != nil {
		return fail(fmt.Errorf("create backoffice HTTP application: %w", err))
	}
	return registry, nil
}

func (registry *Registry) Run(ctx context.Context) error {
	if registry == nil || registry.server == nil {
		return errors.New("backoffice registry is not configured")
	}
	return registry.server.Run(ctx)
}

func (registry *Registry) Close() {
	if registry != nil {
		registry.cleanups.Run()
	}
}

type cleanupStack struct{ functions []func() }

func (stack *cleanupStack) Add(cleanup func()) {
	if stack != nil && cleanup != nil {
		stack.functions = append(stack.functions, cleanup)
	}
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
