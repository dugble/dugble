package worker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	sentrymonitoring "github.com/dugble/dugble/server/internal/adapters/monitoring/sentry"
	natsadapter "github.com/dugble/dugble/server/internal/adapters/nats"
	"github.com/dugble/dugble/server/internal/adapters/postgres"
	"github.com/dugble/dugble/server/internal/config"
	"github.com/dugble/dugble/server/internal/platform/outbox"
)

// Registry owns the worker's long-lived infrastructure and process lifecycle.
type Registry struct {
	config    *config.Config
	postgres  *pgxpool.Pool
	messaging *natsadapter.Client
	outbox    *outbox.Repository
	worker    *Worker
	cleanups  cleanupStack
}

// Start wires and runs the worker until an interrupt or termination signal.
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

// New constructs the worker registry and background modules.
func New(ctx context.Context) (*Registry, error) {
	if ctx == nil {
		return nil, errors.New("worker registry context is required")
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

	startupCtx, cancelStartup := context.WithTimeout(ctx, 15*time.Second)
	defer cancelStartup()

	registry.postgres, err = postgres.New(startupCtx, cfg.DatabaseURL)
	if err != nil {
		return fail(fmt.Errorf("initialize PostgreSQL: %w", err))
	}
	registry.cleanups.Add(registry.postgres.Close)

	registry.messaging, err = natsadapter.New(startupCtx, cfg.NATSURL, "dugble-worker")
	if err != nil {
		return fail(fmt.Errorf("initialize JetStream: %w", err))
	}
	registry.cleanups.Add(func() {
		if closeErr := registry.messaging.Close(); closeErr != nil {
			sentrymonitoring.Warn("close JetStream client", "error", closeErr)
		}
	})
	if err := registry.messaging.Provision(startupCtx, natsadapter.DefaultStreamLimits()); err != nil {
		return fail(fmt.Errorf("provision JetStream topology: %w", err))
	}

	registry.outbox = outbox.NewRepository(registry.postgres)
	modules, err := registry.newModules(startupCtx)
	if err != nil {
		return fail(err)
	}
	registry.worker, err = newWorker(newJobs(modules)...)
	if err != nil {
		return fail(fmt.Errorf("create worker application: %w", err))
	}
	return registry, nil
}

func (registry *Registry) Run(ctx context.Context) error {
	if registry == nil || registry.worker == nil {
		return errors.New("worker registry is not configured")
	}
	return registry.worker.Run(ctx)
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
