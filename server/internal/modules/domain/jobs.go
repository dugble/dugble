package domain

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	sentrymonitoring "github.com/dugble/dugble/server/internal/adapters/monitoring/sentry"
)

type JobConfig struct {
	PollInterval           time.Duration
	PendingCheckInterval   time.Duration
	BatchSize              int32
	Concurrency            int
	LockTimeout            time.Duration
	CheckTimeout           time.Duration
	HealthCheckInterval    time.Duration
	HealthRetryInterval    time.Duration
	HealthFailureThreshold int32
}

func DefaultJobConfig() JobConfig {
	return JobConfig{
		PollInterval:           30 * time.Second,
		PendingCheckInterval:   30 * time.Second,
		BatchSize:              25,
		Concurrency:            5,
		LockTimeout:            2 * time.Minute,
		CheckTimeout:           20 * time.Second,
		HealthCheckInterval:    24 * time.Hour,
		HealthRetryInterval:    time.Hour,
		HealthFailureThreshold: 3,
	}
}

func (config JobConfig) validate() error {
	if config.PollInterval <= 0 || config.PendingCheckInterval <= 0 || config.BatchSize <= 0 || config.Concurrency <= 0 ||
		config.LockTimeout <= 0 || config.CheckTimeout <= 0 || config.HealthCheckInterval <= 0 ||
		config.HealthRetryInterval <= 0 || config.HealthFailureThreshold <= 0 {
		return ErrInvalidJobConfig
	}
	return nil
}

type Job struct {
	repository *Repository
	service    *ReconciliationService
	config     JobConfig
	workerID   string
	now        func() time.Time
}

func NewJob(repository *Repository, checker reconciliationChecker, config JobConfig, workerID string) (*Job, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}
	workerID = strings.TrimSpace(workerID)
	if workerID == "" {
		return nil, ErrWorkerIDRequired
	}
	if repository == nil || checker == nil {
		return nil, ErrJobNotConfigured
	}
	return &Job{
		repository: repository,
		service:    NewReconciliationService(repository, checker, config, workerID),
		config:     config,
		workerID:   workerID,
		now:        func() time.Time { return time.Now().UTC() },
	}, nil
}

func (job *Job) WithNotifier(notifier statusNotifier) *Job {
	if job != nil && job.service != nil {
		job.service.WithNotifier(notifier)
	}
	return job
}

func (job *Job) Run(ctx context.Context) error {
	if job == nil || job.repository == nil || job.service == nil {
		return ErrJobNotConfigured
	}
	ticker := time.NewTicker(job.config.PollInterval)
	defer ticker.Stop()
	for {
		if err := job.poll(ctx); err != nil && !errors.Is(err, context.Canceled) {
			sentrymonitoring.Error("sender domain reconciliation poll failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (job *Job) poll(ctx context.Context) error {
	now := job.now()
	claims, err := job.repository.ClaimPendingReconciliations(
		ctx,
		job.workerID,
		job.config.BatchSize,
		now.Add(-job.config.LockTimeout),
	)
	if err != nil {
		return err
	}
	job.service.now = job.now

	semaphore := make(chan struct{}, job.config.Concurrency)
	var wait sync.WaitGroup
	for _, claim := range claims {
		claim := claim
		wait.Add(1)
		go func() {
			defer wait.Done()
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				return
			}
			if err := job.service.Process(ctx, claim); err != nil && !errors.Is(err, context.Canceled) {
				sentrymonitoring.Error(
					"sender domain reconciliation failed",
					"domain_id", claim.Domain.ID,
					"attempt", claim.Attempt,
					"error", err,
				)
			}
		}()
	}
	wait.Wait()
	return ctx.Err()
}

func nextVerificationDelay(status string, checkErr error, attempt int32, id uuid.UUID, config JobConfig) time.Duration {
	if checkErr != nil {
		return nextCheckDelay(max(attempt-1, 0), id)
	}
	if status == StatusVerified {
		return jitter(config.HealthCheckInterval, id)
	}
	return jitter(config.PendingCheckInterval, id)
}

func nextCheckDelay(attempt int32, id uuid.UUID) time.Duration {
	var delay time.Duration
	switch attempt {
	case 0, 1:
		delay = time.Minute
	case 2:
		delay = 2 * time.Minute
	case 3:
		delay = 5 * time.Minute
	case 4:
		delay = 10 * time.Minute
	case 5:
		delay = 30 * time.Minute
	default:
		delay = time.Hour << min(attempt-6, 2)
		if delay > 6*time.Hour {
			delay = 6 * time.Hour
		}
	}
	return jitter(delay, id)
}

func jitter(delay time.Duration, id uuid.UUID) time.Duration {
	jitterPercent := int(id[0])%21 - 10
	return delay + time.Duration(jitterPercent)*delay/100
}
