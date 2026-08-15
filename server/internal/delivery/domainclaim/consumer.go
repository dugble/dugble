package domainclaimreconciliation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	sentrymonitoring "github.com/dugble/dugble/server/internal/adapters/monitoring/sentry"
	domainclaim "github.com/dugble/dugble/server/internal/modules/domainclaim"
)

type Config struct {
	PollInterval  time.Duration
	BatchSize     int32
	Concurrency   int
	LockTimeout   time.Duration
	HandleTimeout time.Duration
}

func DefaultConfig() Config {
	return Config{
		PollInterval:  30 * time.Second,
		BatchSize:     25,
		Concurrency:   5,
		LockTimeout:   2 * time.Minute,
		HandleTimeout: 60 * time.Second,
	}
}

type Consumer struct {
	repository *domainclaim.Repository
	service    *domainclaim.Service
	config     Config
	workerID   string
	now        func() time.Time
}

func NewConsumer(repository *domainclaim.Repository, service *domainclaim.Service, config Config, workerID string) (*Consumer, error) {
	if repository == nil || service == nil {
		return nil, errors.New("domain claim reconciliation is not configured")
	}
	if strings.TrimSpace(workerID) == "" {
		return nil, errors.New("domain claim reconciliation worker id is required")
	}
	if config.PollInterval <= 0 || config.BatchSize <= 0 || config.Concurrency <= 0 || config.LockTimeout <= 0 || config.HandleTimeout <= 0 {
		return nil, errors.New("invalid domain claim reconciliation config")
	}
	return &Consumer{
		repository: repository,
		service: service,
		config: config,
		workerID: strings.TrimSpace(workerID),
		now: func() time.Time { return time.Now().UTC() },
	}, nil
}

func (c *Consumer) Run(ctx context.Context) error {
	if c == nil {
		return errors.New("domain claim reconciliation consumer is not configured")
	}
	ticker := time.NewTicker(c.config.PollInterval)
	defer ticker.Stop()
	for {
		if err := c.poll(ctx); err != nil && !errors.Is(err, context.Canceled) {
			sentrymonitoring.Error("domain claim reconciliation poll failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (c *Consumer) poll(ctx context.Context) error {
	claims, err := c.repository.ClaimPending(ctx, c.workerID, c.config.BatchSize, c.now().Add(-c.config.LockTimeout))
	if err != nil {
		return err
	}
	semaphore := make(chan struct{}, c.config.Concurrency)
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
			handleCtx, cancel := context.WithTimeout(ctx, c.config.HandleTimeout)
			defer cancel()
			if _, reconcileErr := c.service.Reconcile(handleCtx, claimed.Claim, c.workerID); reconcileErr != nil {
				claimID, parseErr := uuid.Parse(claimed.Claim.ID)
				if parseErr == nil {
					_, _ = c.repository.Release(context.WithoutCancel(ctx), claimID, c.workerID)
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
