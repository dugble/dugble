package outbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type relayStoreStub struct {
	claim       func(context.Context) ([]Event, error)
	marked      []uuid.UUID
	released    []uuid.UUID
	releaseTime time.Time
	lastError   string
}

func (store *relayStoreStub) ClaimBatch(ctx context.Context, _ string, _ int, _ time.Time) ([]Event, error) {
	return store.claim(ctx)
}

func (store *relayStoreStub) MarkPublished(_ context.Context, id uuid.UUID, _ string) error {
	store.marked = append(store.marked, id)
	return nil
}

func (store *relayStoreStub) Release(_ context.Context, id uuid.UUID, _ string, next time.Time, message string) error {
	store.released = append(store.released, id)
	store.releaseTime = next
	store.lastError = message
	return nil
}

type relayPublisherStub struct{ err error }

func (publisher relayPublisherStub) Publish(context.Context, string, []byte, map[string]string, string) error {
	return publisher.err
}

func TestRelayRetriesTransientClaimFailure(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	store := &relayStoreStub{claim: func(context.Context) ([]Event, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("database temporarily unavailable")
		}
		cancel()
		return nil, nil
	}}
	relay := NewRelay(store, relayPublisherStub{}, Config{
		PollInterval: time.Hour, FailureRetryMin: time.Millisecond, FailureRetryMax: time.Millisecond,
	})

	if err := relay.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if calls != 2 {
		t.Fatalf("ClaimBatch calls = %d, want 2", calls)
	}
}

func TestRelayReleasesFailedPublicationForRetry(t *testing.T) {
	t.Parallel()

	event := Event{ID: uuid.New(), AggregateID: uuid.New(), Attempts: 3}
	store := &relayStoreStub{claim: func(context.Context) ([]Event, error) { return nil, nil }}
	relay := NewRelay(store, relayPublisherStub{err: errors.New("broker unavailable")}, Config{})
	before := time.Now().UTC().Add(4 * time.Second)

	if err := relay.processEvent(context.Background(), event); err != nil {
		t.Fatalf("processEvent() error = %v", err)
	}
	if len(store.released) != 1 || store.released[0] != event.ID {
		t.Fatalf("released events = %v, want %s", store.released, event.ID)
	}
	if store.releaseTime.Before(before) {
		t.Fatalf("next attempt = %v, want at least %v", store.releaseTime, before)
	}
	if store.lastError != "broker unavailable" {
		t.Fatalf("last error = %q", store.lastError)
	}
}

func TestFailureBackoffIsBounded(t *testing.T) {
	t.Parallel()

	for attempt, want := range []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 5 * time.Second, 5 * time.Second} {
		if got := failureBackoff(attempt+1, time.Second, 5*time.Second); got != want {
			t.Fatalf("failureBackoff(%d) = %v, want %v", attempt+1, got, want)
		}
	}
}
