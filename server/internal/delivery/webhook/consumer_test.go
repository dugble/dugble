package webhook

import (
	"context"
	"errors"
	"testing"
	"time"
)

type claimQueueStub struct {
	claim func(context.Context) ([]ClaimedDelivery, error)
}

func (queue claimQueueStub) Claim(ctx context.Context, _ string, _ int32, _ time.Time) ([]ClaimedDelivery, error) {
	return queue.claim(ctx)
}

type deliveryProcessorStub struct{}

func (deliveryProcessorStub) Handle(context.Context, ClaimedDelivery) error { return nil }

func TestConsumerRetriesTransientClaimFailure(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	queue := claimQueueStub{claim: func(context.Context) ([]ClaimedDelivery, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("database temporarily unavailable")
		}
		cancel()
		return nil, nil
	}}
	consumer := NewConsumer(queue, deliveryProcessorStub{}, ConsumerConfig{
		PollInterval: time.Hour, FailureRetryMin: time.Millisecond, FailureRetryMax: time.Millisecond,
	}, "worker-1")

	if err := consumer.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if calls != 2 {
		t.Fatalf("Claim calls = %d, want 2", calls)
	}
}

func TestConsumerFailureBackoffIsBounded(t *testing.T) {
	t.Parallel()

	for attempt, want := range []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 5 * time.Second, 5 * time.Second} {
		if got := consumerFailureBackoff(attempt+1, time.Second, 5*time.Second); got != want {
			t.Fatalf("consumerFailureBackoff(%d) = %v, want %v", attempt+1, got, want)
		}
	}
}
