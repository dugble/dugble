package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	awsses "github.com/dugble/dugble/server/internal/adapters/amazon/ses"
	"github.com/dugble/dugble/server/internal/adapters/dns/netdns"
	leamoutsms "github.com/dugble/dugble/server/internal/adapters/leamout/sms"
	mnotifyadapter "github.com/dugble/dugble/server/internal/adapters/mnotify"
	mnotifysms "github.com/dugble/dugble/server/internal/adapters/mnotify/sms"
	sentrymonitoring "github.com/dugble/dugble/server/internal/adapters/monitoring/sentry"
	"github.com/dugble/dugble/server/internal/adapters/moolre"
	moolresender "github.com/dugble/dugble/server/internal/adapters/moolre/sender"
	moolresms "github.com/dugble/dugble/server/internal/adapters/moolre/sms"
	runnagesms "github.com/dugble/dugble/server/internal/adapters/runnage/sms"
	chargeSubscription "github.com/dugble/dugble/server/internal/billing/charge/subscription"
	platformbilling "github.com/dugble/dugble/server/internal/billing/charge/usage"
	subscriptionLifecycle "github.com/dugble/dugble/server/internal/billing/subscription/lifecycle"
	subscriptionRenewal "github.com/dugble/dugble/server/internal/billing/subscription/renewal"
	broadcastexecution "github.com/dugble/dugble/server/internal/delivery/broadcast"
	emailfeedback "github.com/dugble/dugble/server/internal/delivery/email/feedback"
	emaildelivery "github.com/dugble/dugble/server/internal/delivery/email/outbound"
	systememail "github.com/dugble/dugble/server/internal/delivery/email/system"
	smscampaignexecution "github.com/dugble/dugble/server/internal/delivery/sms/campaign"
	smsfeedback "github.com/dugble/dugble/server/internal/delivery/sms/feedback"
	smsdelivery "github.com/dugble/dugble/server/internal/delivery/sms/outbound"
	webhookdelivery "github.com/dugble/dugble/server/internal/delivery/webhook"
	broadcastmodule "github.com/dugble/dugble/server/internal/modules/broadcast"
	smscampaignmodule "github.com/dugble/dugble/server/internal/modules/campaign"
	domainmodule "github.com/dugble/dugble/server/internal/modules/domain"
	domainclaimmodule "github.com/dugble/dugble/server/internal/modules/domainclaim"
	"github.com/dugble/dugble/server/internal/modules/emailtenant"
	senderidmodule "github.com/dugble/dugble/server/internal/modules/senderid"
	smsmodule "github.com/dugble/dugble/server/internal/modules/sms"
	webhookmodule "github.com/dugble/dugble/server/internal/modules/webhooks"
	platformemail "github.com/dugble/dugble/server/internal/platform/awsses"
	platformevent "github.com/dugble/dugble/server/internal/platform/event"
	"github.com/dugble/dugble/server/internal/platform/outbox"
	platformsms "github.com/dugble/dugble/server/internal/platform/sms"
	"github.com/dugble/dugble/server/internal/platform/systemmail"
	platformwebhook "github.com/dugble/dugble/server/internal/platform/webhook"
)

type modules struct {
	subscriptionRenewal       func(context.Context) error
	outboxRelay               func(context.Context) error
	transactionalEmail        func(context.Context) error
	marketingEmail            func(context.Context) error
	systemEmail               func(context.Context) error
	emailTenantProvisioning   func(context.Context) error
	emailFeedback             func(context.Context) error
	emailFeedbackReconciler   func(context.Context) error
	emailFeedbackMetrics      func(context.Context) error
	smsDelivery               func(context.Context) error
	smsFeedback               func(context.Context) error
	smsCampaign               func(context.Context) error
	webhookDelivery           func(context.Context) error
	domainReconciliation      func(context.Context) error
	domainClaimReconciliation func(context.Context) error
	broadcastExecution        func(context.Context) error
	senderIDReconciliation    func(context.Context) error
}

func (registry *Registry) newModules(startupCtx context.Context) (modules, error) {
	cfg := registry.config
	db := registry.postgres
	messagingClient := registry.messaging
	outboxRepository := registry.outbox
	processedEvents := outboxRepository

	webhookModuleRepository := webhookmodule.NewRepository(db)
	webhookEmitter := platformwebhook.NewEmitter(webhookModuleRepository)
	lifecycleEmitter := webhookEmitter
	broadcastExecutionRepository := broadcastmodule.NewRepositoryWithEventEmitter(
		db,
		platformevent.NewEmitter(platformwebhook.NewEventSink(webhookEmitter)),
	)
	broadcastExecutionJob := broadcastmodule.NewJob(
		broadcastexecution.NewProcessor(broadcastExecutionRepository),
		broadcastmodule.JobConfig{PollInterval: time.Second, BatchSize: 100},
	)

	emailSender, err := awsses.NewSESSender(
		startupCtx,
		cfg.AWS.Region,
		cfg.AWS.FromEmail,
		cfg.AWS.AccessKey,
		cfg.AWS.SecretKey,
		platformemail.TransactionalConfigurationSet,
	)
	if err != nil {
		return modules{}, fmt.Errorf("initialize SES email sender: %w", err)
	}

	emailDeliveryProcessor := emaildelivery.NewProcessor(emaildelivery.NewRepository(db), emailSender)
	transactionalEmailConsumer := emaildelivery.NewConsumer(
		messagingClient,
		processedEvents,
		emailDeliveryProcessor,
		emaildelivery.ConsumerConfig{
			Stream:         "transactional",
			Concurrency:    5,
			AckWait:        2 * time.Minute,
			HandlerTimeout: 45 * time.Second,
			MaxDeliver:     6,
			RetryPolicy:    emaildelivery.DefaultRetryPolicy(),
		},
	)
	marketingEmailConsumer := emaildelivery.NewConsumer(
		messagingClient,
		processedEvents,
		emailDeliveryProcessor,
		emaildelivery.ConsumerConfig{
			Stream:         "marketing",
			Concurrency:    2,
			AckWait:        2 * time.Minute,
			HandlerTimeout: 45 * time.Second,
			MaxDeliver:     6,
			RetryPolicy:    emaildelivery.DefaultRetryPolicy(),
		},
	)
	systemEmailConsumer := systememail.NewConsumer(
		messagingClient,
		processedEvents,
		emailSender,
		systememail.ConsumerConfig{Concurrency: 3, AckWait: time.Minute, HandlerTimeout: 30 * time.Second, MaxDeliver: 6},
	)
	systemEmailRenderer, err := systemmail.NewRenderer()
	if err != nil {
		return modules{}, fmt.Errorf("initialize system email renderer: %w", err)
	}
	systemEmailQueue := systememail.NewQueue(outboxRepository, platformemail.Message{Provider: awsses.ProviderSES, Region: cfg.AWS.Region, Stream: "transactional", ConfigurationSet: platformemail.TransactionalConfigurationSet, SESTenantName: platformemail.SystemSESTenantName})
	notificationEmailService := systemmail.NewEmailService(systemEmailQueue, systemEmailRenderer, cfg.FrontendURL, cfg.AWS.FromEmail)
	emailTenantRepository := emailtenant.NewRepository(db)
	emailTenantService := emailtenant.NewService(db, emailTenantRepository, emailtenant.NewProvisioningQueue(outboxRepository))
	emailTenantConsumer := emailtenant.NewProvisioningConsumer(
		messagingClient,
		processedEvents,
		emailtenant.NewProvisioningProcessor(emailTenantRepository, emailSender),
		emailtenant.ProvisioningConsumerConfig{Concurrency: 3, AckWait: 2 * time.Minute, HandlerTimeout: 60 * time.Second, RetryBackOff: emailtenant.DefaultProvisioningRetryBackOff()},
	)

	feedbackMetrics := emailfeedback.DefaultMetrics
	emailFeedbackRepository := emailfeedback.NewRepositoryWithWebhookEmitter(db, lifecycleEmitter)
	emailFeedbackConsumer := emailfeedback.NewConsumer(
		messagingClient,
		processedEvents,
		emailfeedback.NewHandlerWithMetrics(emailFeedbackRepository, feedbackMetrics),
		emailfeedback.ConsumerConfig{
			Concurrency:    5,
			AckWait:        time.Minute,
			HandlerTimeout: 30 * time.Second,
			MaxDeliver:     6,
			RetryPolicy:    emailfeedback.DefaultRetryPolicy(),
		},
	)
	emailFeedbackReconciler := emailfeedback.NewObservedReconciler(
		emailFeedbackRepository,
		emailfeedback.ReconcilerConfig{
			PollInterval:  5 * time.Second,
			BatchSize:     25,
			Concurrency:   5,
			LeaseDuration: 2 * time.Minute,
			HandleTimeout: 30 * time.Second,
		},
		feedbackMetrics,
	)
	emailFeedbackMetricsCollector := emailfeedback.NewMetricsCollector(db, feedbackMetrics, 15*time.Second)

	dnsVerifier := netdns.New()
	domainRepository := domainmodule.NewRepository(db)
	domainService := domainmodule.NewService(domainRepository, emailSender, dnsVerifier).WithDatabase(db)
	domainJob, err := domainmodule.NewJob(
		db,
		domainRepository,
		domainService,
		domainmodule.DefaultJobConfig(),
		"sender-domain-reconciliation-"+uuid.NewString(),
	)
	if err != nil {
		return modules{}, fmt.Errorf("initialize sender domain reconciliation: %w", err)
	}
	domainJob.WithNotifier(notificationEmailService)

	domainClaimRepository := domainclaimmodule.NewRepository(db)
	domainClaimService := domainclaimmodule.NewService(db, domainClaimRepository, emailSender, dnsVerifier, emailTenantService)
	domainClaimJob, err := domainclaimmodule.NewJob(
		domainClaimRepository,
		domainClaimService,
		domainclaimmodule.DefaultJobConfig(),
		"sender-domain-claim-reconciliation-"+uuid.NewString(),
	)
	if err != nil {
		return modules{}, fmt.Errorf("initialize domain claim reconciliation: %w", err)
	}

	senderIDRun := disabledSenderIDReconciliation
	if cfg.Moolre.VASKey == "" {
		sentrymonitoring.Warn("Moolre Sender ID reconciliation is disabled because MOOLRE_VAS_KEY is empty")
	} else {
		senderIDProvider := moolresender.NewProvider(moolre.NewClient(cfg.Moolre.VASKey))
		senderIDJob, jobErr := senderidmodule.NewJob(
			senderidmodule.NewRepository(db),
			senderidmodule.DefaultJobConfig(),
			"sender-id-reconciliation-"+uuid.NewString(),
			senderIDProvider,
		)
		if jobErr != nil {
			return modules{}, fmt.Errorf("initialize Sender ID reconciliation: %w", jobErr)
		}
		senderIDJob.WithNotifier(notificationEmailService)
		senderIDRun = senderIDJob.Run
	}

	smsRouter, err := platformsms.NewRoutingService(
		platformsms.DefaultRoutingConfig(),
		mnotifysms.NewProvider(mnotifyadapter.NewClient(cfg.MNotify.APIKey)),
		moolresms.NewProvider(moolre.NewClient(cfg.Moolre.VASKey)),
		leamoutsms.NewProvider(),
		runnagesms.NewProvider(),
	)
	if err != nil {
		return modules{}, fmt.Errorf("initialize SMS router: %w", err)
	}
	smsSender, err := platformsms.NewService(smsRouter)
	if err != nil {
		return modules{}, fmt.Errorf("initialize SMS sender: %w", err)
	}
	smsRepository := smsmodule.NewRepositoryWithWebhookEmitter(db, lifecycleEmitter)
	smsCampaignSMSService := smsmodule.NewService(
		smsRepository,
		nil,
		smsdelivery.NewQueue(outboxRepository),
		platformbilling.NewService(platformbilling.NewRepository(db)),
	)
	smsCampaignJob := smscampaignmodule.NewJob(
		smscampaignexecution.NewProcessor(smscampaignmodule.NewRepository(db), smsCampaignSMSService),
		smscampaignmodule.JobConfig{PollInterval: time.Second, BatchSize: 100},
	)
	smsConsumer := smsdelivery.NewConsumer(
		messagingClient,
		processedEvents,
		smsdelivery.NewProcessor(smsdelivery.NewRepository(db, smsRepository), smsSender),
		smsdelivery.ConsumerConfig{Concurrency: 10, AckWait: 2 * time.Minute, HandlerTimeout: 45 * time.Second, MaxDeliver: 6},
	)
	smsFeedbackRepository := smsfeedback.NewRepositoryWithMessageRepository(db, smsRepository)
	smsFeedbackProcessor := smsfeedback.NewProcessor(smsFeedbackRepository)
	smsFeedbackReconciler := smsfeedback.NewReconciler(
		smsFeedbackRepository,
		smsSender,
		smsFeedbackProcessor,
		smsfeedback.ReconcilerConfig{BatchSize: 100, Concurrency: 10, ProviderTimeout: 15 * time.Second},
	)
	smsFeedbackConsumer := smsfeedback.NewConsumer(smsFeedbackReconciler, smsfeedback.ConsumerConfig{PollInterval: 30 * time.Second})

	outboxRelay := outbox.NewRelay(
		outboxRepository,
		messagingClient,
		outbox.Config{PollInterval: 500 * time.Millisecond, BatchSize: 100, LockTimeout: 30 * time.Second},
	)
	webhookWorkerID := "webhook-delivery-" + uuid.NewString()
	webhookRepository := webhookdelivery.NewRepository(db, webhookdelivery.RepositoryConfig{AutoDisableAfter: 20})
	webhookProcessor := webhookdelivery.NewProcessor(
		webhookRepository,
		webhookdelivery.NewClient(10*time.Second),
		webhookdelivery.DefaultRetryPolicy(),
		webhookWorkerID,
	).WithNotifier(notificationEmailService)
	webhookConsumer := webhookdelivery.NewConsumer(
		webhookRepository,
		webhookProcessor,
		webhookdelivery.ConsumerConfig{PollInterval: 500 * time.Millisecond, BatchSize: 50, Concurrency: 10, LockTimeout: 30 * time.Second, HandleTimeout: 15 * time.Second},
		webhookWorkerID,
	)

	renewalService := subscriptionRenewal.NewService(
		subscriptionRenewal.NewRepository(),
		chargeSubscription.NewService(chargeSubscription.NewRepository()),
		subscriptionLifecycle.NewService(),
	).WithEventPublisher(subscriptionRenewal.NewEventPublisher(outboxRepository))
	renewalService.WithPastDueNotifier(notificationEmailService)
	renewalService.WithPlanChangeNotifier(notificationEmailService)
	renewalConfig := subscriptionRenewal.DefaultConfig()
	renewalConfig.OnFailure = func(ctx context.Context, failure subscriptionRenewal.Failure) {
		sentrymonitoring.ErrorContext(ctx, "subscription renewal failed", "team_id", failure.TeamID, "error", failure.Err)
	}
	renewalWorker, err := subscriptionRenewal.NewWorker(db, renewalService, renewalConfig)
	if err != nil {
		return modules{}, fmt.Errorf("initialize subscription renewal worker: %w", err)
	}

	return modules{
		subscriptionRenewal:       renewalWorker.Run,
		outboxRelay:               outboxRelay.Run,
		transactionalEmail:        transactionalEmailConsumer.Run,
		marketingEmail:            marketingEmailConsumer.Run,
		systemEmail:               systemEmailConsumer.Run,
		emailTenantProvisioning:   emailTenantConsumer.Run,
		emailFeedback:             emailFeedbackConsumer.Run,
		emailFeedbackReconciler:   emailFeedbackReconciler.Run,
		emailFeedbackMetrics:      emailFeedbackMetricsCollector.Run,
		smsDelivery:               smsConsumer.Run,
		smsFeedback:               smsFeedbackConsumer.Run,
		smsCampaign:               smsCampaignJob.Run,
		webhookDelivery:           webhookConsumer.Run,
		domainReconciliation:      domainJob.Run,
		domainClaimReconciliation: domainClaimJob.Run,
		broadcastExecution:        broadcastExecutionJob.Run,
		senderIDReconciliation:    senderIDRun,
	}, nil
}

func disabledSenderIDReconciliation(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}
