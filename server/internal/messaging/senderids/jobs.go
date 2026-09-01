package senderid

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	sentrymonitoring "github.com/dugble/dugble/server/internal/integrations/monitoring/sentry"
	platformsenderid "github.com/dugble/dugble/server/internal/messaging/senderids/provider"
)

type JobConfig struct {
	PollInterval         time.Duration
	BatchSize            int32
	Concurrency          int
	LockTimeout          time.Duration
	ProviderTimeout      time.Duration
	PendingCheckInterval time.Duration
	RetryBaseInterval    time.Duration
	MaxRetryInterval     time.Duration
}

func DefaultJobConfig() JobConfig {
	return JobConfig{
		PollInterval:         15 * time.Second,
		BatchSize:            25,
		Concurrency:          5,
		LockTimeout:          2 * time.Minute,
		ProviderTimeout:      20 * time.Second,
		PendingCheckInterval: 2 * time.Minute,
		RetryBaseInterval:    30 * time.Second,
		MaxRetryInterval:     time.Hour,
	}
}

func (config JobConfig) validate() error {
	if config.PollInterval <= 0 || config.BatchSize <= 0 || config.Concurrency <= 0 ||
		config.LockTimeout <= 0 || config.ProviderTimeout <= 0 || config.PendingCheckInterval <= 0 ||
		config.RetryBaseInterval <= 0 || config.MaxRetryInterval < config.RetryBaseInterval {
		return ErrInvalidJobConfig
	}
	return nil
}

type Job struct {
	repository *Repository
	service    *ReconciliationService
	providers  map[string]platformsenderid.Provider
	config     JobConfig
	workerID   string
	now        func() time.Time
}

func NewJob(repository *Repository, config JobConfig, workerID string, providers ...platformsenderid.Provider) (*Job, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}
	workerID = strings.TrimSpace(workerID)
	if workerID == "" {
		return nil, ErrWorkerIDRequired
	}
	registry := make(map[string]platformsenderid.Provider, len(providers))
	for _, provider := range providers {
		if provider == nil {
			return nil, errors.New("sender ID provider is required")
		}
		providerID := strings.ToLower(strings.TrimSpace(provider.ID()))
		if providerID == "" {
			return nil, errors.New("sender ID provider ID is required")
		}
		if _, exists := registry[providerID]; exists {
			return nil, fmt.Errorf("duplicate Sender ID provider %q", providerID)
		}
		registry[providerID] = provider
	}
	if len(registry) == 0 {
		return nil, errors.New("at least one Sender ID provider is required")
	}
	return &Job{
		repository: repository,
		service:    NewReconciliationService(repository, config, workerID),
		providers:  registry,
		config:     config,
		workerID:   workerID,
		now:        func() time.Time { return time.Now().UTC() },
	}, nil
}

func (job *Job) WithNotifier(notifier reconciliationNotifier) *Job {
	job.service.WithNotifier(notifier)
	return job
}

func (job *Job) Run(ctx context.Context) error {
	if job == nil || job.repository == nil || job.service == nil || len(job.providers) == 0 {
		return ErrJobNotConfigured
	}
	ticker := time.NewTicker(job.config.PollInterval)
	defer ticker.Stop()
	for {
		if err := job.poll(ctx); err != nil && !errors.Is(err, context.Canceled) {
			sentrymonitoring.Error("Sender ID reconciliation poll failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

type workItem struct {
	provider platformsenderid.Provider
	claim    RegistrationClaim
}

func (job *Job) poll(ctx context.Context) error {
	now := job.now()
	items := make([]workItem, 0, int(job.config.BatchSize)*len(job.providers))
	var joined error
	for providerID, provider := range job.providers {
		claims, err := job.repository.ClaimPendingRegistrations(ctx, job.workerID, providerID, job.config.BatchSize, now.Add(-job.config.LockTimeout))
		if err != nil {
			joined = errors.Join(joined, fmt.Errorf("claim %s Sender ID registrations: %w", providerID, err))
			continue
		}
		for _, claim := range claims {
			items = append(items, workItem{provider: provider, claim: claim})
		}
	}

	semaphore := make(chan struct{}, job.config.Concurrency)
	var wait sync.WaitGroup
	var mutex sync.Mutex
	for _, item := range items {
		item := item
		wait.Add(1)
		go func() {
			defer wait.Done()
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				return
			}
			if err := job.service.Process(ctx, item.provider, item.claim); err != nil && !errors.Is(err, context.Canceled) {
				sentrymonitoring.Error("Sender ID reconciliation failed", "sender_id", item.claim.ID, "provider", item.claim.Provider, "attempt", item.claim.Attempt, "error", err)
				mutex.Lock()
				joined = errors.Join(joined, err)
				mutex.Unlock()
			}
		}()
	}
	wait.Wait()
	return joined
}
