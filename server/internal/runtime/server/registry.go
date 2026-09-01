package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/arcjet/arcjet-go"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	paymentmodule "github.com/dugble/dugble/server/internal/commercial/payments"
	"github.com/dugble/dugble/server/internal/delivery/email/feedback"
	systememail "github.com/dugble/dugble/server/internal/delivery/email/system"
	awsses "github.com/dugble/dugble/server/internal/integrations/amazon/ses"
	awssns "github.com/dugble/dugble/server/internal/integrations/amazon/sns"
	"github.com/dugble/dugble/server/internal/integrations/hubtel"
	leamoutsms "github.com/dugble/dugble/server/internal/integrations/leamout/sms"
	mnotifyadapter "github.com/dugble/dugble/server/internal/integrations/mnotify"
	mnotifysms "github.com/dugble/dugble/server/internal/integrations/mnotify/sms"
	newrelicmonitoring "github.com/dugble/dugble/server/internal/integrations/monitoring/newrelic"
	sentrymonitoring "github.com/dugble/dugble/server/internal/integrations/monitoring/sentry"
	"github.com/dugble/dugble/server/internal/integrations/moolre"
	moolresms "github.com/dugble/dugble/server/internal/integrations/moolre/sms"
	"github.com/dugble/dugble/server/internal/integrations/postgres"
	redisadapter "github.com/dugble/dugble/server/internal/integrations/redis"
	runnagesms "github.com/dugble/dugble/server/internal/integrations/runnage/sms"
	arcjetadapter "github.com/dugble/dugble/server/internal/integrations/security/arcjet"
	platformemail "github.com/dugble/dugble/server/internal/messaging/email/provider"
	platformsms "github.com/dugble/dugble/server/internal/messaging/sms/provider"
	"github.com/dugble/dugble/server/internal/modules/outbox"
	"github.com/dugble/dugble/server/internal/platform/config"
	"github.com/dugble/dugble/server/internal/platform/idempotency"
	httptransport "github.com/dugble/dugble/server/internal/runtime/http"
	httpmiddleware "github.com/dugble/dugble/server/internal/runtime/middleware"
	providersns "github.com/dugble/dugble/server/internal/runtime/provider/aws/sns"
)

// Registry owns the server's long-lived infrastructure and HTTP lifecycle.
type Registry struct {
	config         *config.Config
	postgres       *pgxpool.Pool
	redis          *redis.Client
	arcjet         *arcjet.Client
	emailClient    *awsses.Client
	smsSender      *platformsms.Service
	outbox         *outbox.Repository
	providerSNS    *providersns.Handler
	hubtelProvider *hubtel.Provider
	hubtelPayments *paymentmodule.Service
	server         *httptransport.Server
	cleanups       cleanupStack
}

// Start wires and runs the server until an interrupt or termination signal.
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

// New constructs the server registry and all long-lived infrastructure.
func New(ctx context.Context) (*Registry, error) {
	if ctx == nil {
		return nil, errors.New("server registry context is required")
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

	newRelic, err := newrelicmonitoring.New("dugble-api", cfg.AppEnv, cfg.NewRelic)
	if err != nil {
		return fail(fmt.Errorf("initialize New Relic client: %w", err))
	}
	registry.cleanups.Add(func() { newrelicmonitoring.Shutdown(newRelic, 5*time.Second) })

	startupCtx, cancelStartup := context.WithTimeout(ctx, 15*time.Second)
	defer cancelStartup()

	registry.postgres, err = postgres.New(startupCtx, cfg.DatabaseURL)
	if err != nil {
		return fail(fmt.Errorf("initialize PostgreSQL: %w", err))
	}
	registry.cleanups.Add(registry.postgres.Close)

	registry.redis, err = redisadapter.New(startupCtx, cfg.RedisURL)
	if err != nil {
		return fail(fmt.Errorf("initialize Redis: %w", err))
	}
	registry.cleanups.Add(func() {
		if closeErr := registry.redis.Close(); closeErr != nil {
			sentrymonitoring.Warn("close Redis client", "error", closeErr)
		}
	})

	registry.arcjet, err = arcjetadapter.New(cfg.ArcjetKey)
	if err != nil {
		return fail(fmt.Errorf("initialize Arcjet client: %w", err))
	}
	registry.emailClient, err = newEmailClient(cfg)
	if err != nil {
		return fail(fmt.Errorf("initialize email client: %w", err))
	}
	registry.outbox = outbox.NewRepository(registry.postgres)
	registry.providerSNS = newProviderSNSHandler(cfg, registry.postgres, registry.outbox)
	registry.hubtelProvider, registry.hubtelPayments = newHubtelServices(cfg, registry.postgres)
	registry.smsSender, err = newSMSSender(cfg)
	if err != nil {
		return fail(err)
	}

	router, err := httptransport.NewRouter(registry.routerConfig(), registry.registerRoutes)
	if err != nil {
		return fail(fmt.Errorf("create HTTP router: %w", err))
	}
	registry.server, err = httptransport.NewServer(newrelicmonitoring.WrapHTTP(newRelic, router), ":"+cfg.HTTPPort)
	if err != nil {
		return fail(fmt.Errorf("create HTTP application: %w", err))
	}
	return registry, nil
}

func (registry *Registry) Run(ctx context.Context) error {
	if registry == nil || registry.server == nil {
		return errors.New("server registry is not configured")
	}
	return registry.server.Run(ctx)
}

func (registry *Registry) Close() {
	if registry != nil {
		registry.cleanups.Run()
	}
}

func (registry *Registry) routerConfig() httptransport.RouterConfig {
	return httptransport.RouterConfig{
		Development: registry.config.IsDevelopment(),
		CORSOrigins: registry.config.CORSOrigins,
		Arcjet:      registry.arcjet,
		BodyLimit:   platformemail.MaxHTTPRequestBytes,
		Idempotency: httpmiddleware.IdempotencyConfig{Repository: idempotency.NewRepository(registry.postgres)},
		Middleware:  defaultHTTPMiddleware(),
	}
}

func newEmailClient(cfg *config.Config) (*awsses.Client, error) {
	return awsses.NewClient(cfg.AWS.Region, cfg.AWS.FromEmail, cfg.AWS.AccessKey, cfg.AWS.SecretKey, platformemail.TransactionalConfigurationSet)
}

func newHubtelServices(cfg *config.Config, db *pgxpool.Pool) (*hubtel.Provider, *paymentmodule.Service) {
	if !cfg.Hubtel.Enabled {
		return nil, nil
	}
	provider := hubtel.NewProvider(hubtel.NewClient(cfg.Hubtel))
	return provider, paymentmodule.NewService(paymentmodule.NewRepository(db))
}

func newSystemEmailQueue(cfg *config.Config, repository *outbox.Repository) *systememail.Queue {
	return systememail.NewQueue(repository, platformemail.Message{Provider: awsses.ProviderSES, Region: cfg.AWS.Region, Stream: "transactional", ConfigurationSet: platformemail.TransactionalConfigurationSet, SESTenantName: platformemail.SystemSESTenantName})
}

func newProviderSNSHandler(cfg *config.Config, db *pgxpool.Pool, repository *outbox.Repository) *providersns.Handler {
	if len(cfg.AWS.SNSTopicARNs) == 0 {
		return nil
	}
	verifier := awssns.NewVerifier(cfg.AWS.SNSTopicARNs, awssns.NewHTTPCertificateLoader(nil))
	confirmer := awssns.NewConfirmer(awssns.NewHTTPConfirmSubscriptionClient(nil))
	return providersns.NewHandler(verifier, confirmer, feedback.NewRepository(db, repository))
}

func newSMSSender(cfg *config.Config) (*platformsms.Service, error) {
	router, err := platformsms.NewRoutingService(platformsms.DefaultRoutingConfig(), mnotifysms.NewProvider(mnotifyadapter.NewClient(cfg.MNotify.APIKey)), moolresms.NewProvider(moolre.NewClient(cfg.Moolre.VASKey)), leamoutsms.NewProvider(), runnagesms.NewProvider())
	if err != nil {
		return nil, fmt.Errorf("initialize SMS router: %w", err)
	}
	sender, err := platformsms.NewService(router)
	if err != nil {
		return nil, fmt.Errorf("initialize SMS sender: %w", err)
	}
	return sender, nil
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
