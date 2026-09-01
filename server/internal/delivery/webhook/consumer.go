package webhook

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	sentrymonitoring "github.com/dugble/dugble/server/internal/integrations/monitoring/sentry"
)

type deliveryProcessor interface {
	Handle(context.Context, ClaimedDelivery) error
}

type ConsumerConfig struct {
	PollInterval    time.Duration
	BatchSize       int32
	Concurrency     int
	LockTimeout     time.Duration
	HandleTimeout   time.Duration
	FailureRetryMin time.Duration
	FailureRetryMax time.Duration
}

type Consumer struct {
	queue     ClaimQueue
	processor deliveryProcessor
	config    ConsumerConfig
	workerID  string
}

func NewConsumer(queue ClaimQueue, processor deliveryProcessor, config ConsumerConfig, workerID string) *Consumer {
	if config.PollInterval <= 0 {
		config.PollInterval = 500 * time.Millisecond
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 50
	}
	if config.Concurrency <= 0 {
		config.Concurrency = 10
	}
	if config.LockTimeout <= 0 {
		config.LockTimeout = 30 * time.Second
	}
	if config.HandleTimeout <= 0 {
		config.HandleTimeout = 15 * time.Second
	}
	if config.FailureRetryMin <= 0 {
		config.FailureRetryMin = time.Second
	}
	if config.FailureRetryMax <= 0 {
		config.FailureRetryMax = 30 * time.Second
	}
	if config.FailureRetryMax < config.FailureRetryMin {
		config.FailureRetryMax = config.FailureRetryMin
	}
	return &Consumer{
		queue: queue, processor: processor, config: config,
		workerID: strings.TrimSpace(workerID),
	}
}

func (consumer *Consumer) Run(ctx context.Context) error {
	if consumer == nil || consumer.queue == nil {
		return ErrQueueNotConfigured
	}
	if consumer.processor == nil {
		return ErrProcessorNotConfigured
	}
	if consumer.workerID == "" {
		return errors.New("webhook delivery worker id is required")
	}

	consecutiveFailures := 0
	for {
		processed, err := consumer.processBatch(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			consecutiveFailures++
			delay := consumerFailureBackoff(consecutiveFailures, consumer.config.FailureRetryMin, consumer.config.FailureRetryMax)
			sentrymonitoring.Error("webhook delivery batch failed; retrying", "attempt", consecutiveFailures, "retry_in", delay, "error", err)
			if !waitForConsumerRetry(ctx, delay) {
				return nil
			}
			continue
		}
		consecutiveFailures = 0
		if processed == int(consumer.config.BatchSize) {
			continue
		}

		timer := time.NewTimer(consumer.config.PollInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil
		case <-timer.C:
		}
	}
}

func consumerFailureBackoff(attempt int, minimum, maximum time.Duration) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := minimum
	for range min(attempt-1, 30) {
		if delay >= maximum || delay > maximum/2 {
			return maximum
		}
		delay *= 2
	}
	return min(delay, maximum)
}

func waitForConsumerRetry(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (consumer *Consumer) processBatch(ctx context.Context) (int, error) {
	staleBefore := time.Now().UTC().Add(-consumer.config.LockTimeout)
	deliveries, err := consumer.queue.Claim(ctx, consumer.workerID, consumer.config.BatchSize, staleBefore)
	if err != nil {
		return 0, fmt.Errorf("claim webhook deliveries: %w", err)
	}
	if len(deliveries) == 0 {
		return 0, nil
	}

	semaphore := make(chan struct{}, consumer.config.Concurrency)
	var group sync.WaitGroup

dispatchLoop:
	for _, delivery := range deliveries {
		select {
		case semaphore <- struct{}{}:
		case <-ctx.Done():
			break dispatchLoop
		}
		group.Add(1)
		go func(delivery ClaimedDelivery) {
			defer group.Done()
			defer func() { <-semaphore }()

			handleContext, cancel := context.WithTimeout(ctx, consumer.config.HandleTimeout)
			defer cancel()
			if err := consumer.processor.Handle(handleContext, delivery); err != nil && ctx.Err() == nil {
				sentrymonitoring.Error(
					"webhook delivery failed",
					"delivery_id", delivery.ID,
					"event_id", delivery.EventID,
					"endpoint_id", delivery.EndpointID,
					"attempt", delivery.AttemptCount,
					"error", err,
				)
			}
		}(delivery)
	}
	group.Wait()
	return len(deliveries), ctx.Err()
}
