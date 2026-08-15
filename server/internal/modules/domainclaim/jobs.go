package domainclaim

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	sentrymonitoring "github.com/dugble/dugble/server/internal/adapters/monitoring/sentry"
)

type JobConfig struct {
	PollInterval  time.Duration
	BatchSize     int32
	Concurrency   int
	LockTimeout   time.Duration
	HandleTimeout time.Duration
}

func DefaultJobConfig() JobConfig {
	return JobConfig{
		PollInterval:  30 * time.Second,
		BatchSize:     25,
		Concurrency:   5,
		LockTimeout:   2 * time.Minute,
		HandleTimeout: 60 * time.Second,
	}
}

type Job struct {
	repository *Repository
	service    *Service
	config     JobConfig
	workerID   string
	now        func() time.Time
}

func NewJob(repository *Repository, service *Service, config JobConfig, workerID string) (*Job, error) {
	if repository == nil || service == nil {
		return nil, errors.New("domain claim reconciliation is not configured")
	}
	if strings.TrimSpace(workerID) == "" {
		return nil, errors.New("domain claim reconciliation worker id is required")
	}
	if config.PollInterval <= 0 || config.BatchSize <= 0 || config.Concurrency <= 0 || config.LockTimeout <= 0 || config.HandleTimeout <= 0 {
		return nil, errors.New("invalid domain claim reconciliation config")
	}
	return &Job{
		repository: repository,
		service:    service,
		config:     config,
		workerID:   strings.TrimSpace(workerID),
		now:        func() time.Time { return time.Now().UTC() },
	}, nil
}

func (j *Job) Run(ctx context.Context) error {
	if j == nil {
		return errors.New("domain claim reconciliation job is not configured")
	}
	ticker := time.NewTicker(j.config.PollInterval)
	defer ticker.Stop()
	for {
		if err := j.poll(ctx); err != nil && !errors.Is(err, context.Canceled) {
			sentrymonitoring.Error("domain claim reconciliation poll failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (j *Job) poll(ctx context.Context) error {
	claims, err := j.repository.ClaimPending(ctx, j.workerID, j.config.BatchSize, j.now().Add(-j.config.LockTimeout))
	if err != nil {
		return err
	}
	semaphore := make(chan struct{}, j.config.Concurrency)
	var wait sync.WaitGroup
	for _, claimed := range claims {
		claimed := claimed
		wait.Add(1)
		go func() {
			defer wait.Done()
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				return
			}
			handleCtx, cancel := context.WithTimeout(ctx, j.config.HandleTimeout)
			defer cancel()
			if _, reconcileErr := j.service.Reconcile(handleCtx, claimed.Claim, j.workerID); reconcileErr != nil {
				claimID, parseErr := uuid.Parse(claimed.Claim.ID)
				if parseErr == nil {
					_, _ = j.repository.Release(context.WithoutCancel(ctx), claimID, j.workerID)
				}
				sentrymonitoring.Error(
					"domain claim reconciliation failed",
					"claim_id", claimed.Claim.ID,
					"domain", claimed.Claim.Name,
					"error", reconcileErr,
				)
			}
		}()
	}
	wait.Wait()
	if ctx.Err() != nil {
		return fmt.Errorf("domain claim reconciliation context: %w", ctx.Err())
	}
	return nil
}
