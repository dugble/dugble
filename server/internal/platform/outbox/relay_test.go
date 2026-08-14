package outbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type relayStoreStub struct {
	claim           func(context.Context) ([]Event, error)
	marked          []uuid.UUID
	released        []uuid.UUID
	quarantined     []uuid.UUID
	releaseTime     time.Time
	lastError       string
	quarantineCode  string
	quarantineError string
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

func (store *relayStoreStub) Quarantine(_ context.Context, id uuid.UUID, _ string, code string, reason string) error {
	store.quarantined = append(store.quarantined, id)
	store.quarantineCode = code
	store.quarantineError = reason
	return nil
}

type relayPublisherStub struct{ err error }

func (publisher relayPublisherStub) Publish(context.Context, string, []byte, map[string]string, string) error {
	return publisher.err
}

type permanentTestError struct{ error }

func (permanentTestError) Permanent() bool     { return true }
func (permanentTestError) FailureCode() string { return "invalid_message" }

type retryableTestError struct{ error }

func (retryableTestError) Retryable() bool     { return true }
func (retryableTestError) FailureCode() string { return "broker_unavailable" }

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

	event := Event{ID: uuid.New(), AggregateID: uuid.New(), PublishFailures: 2}
	store := &relayStoreStub{claim: func(context.Context) ([]Event, error) { return nil, nil }}
	relay := NewRelay(store, relayPublisherStub{err: retryableTestError{errors.New("broker unavailable")}}, Config{})
	before := time.Now().UTC().Add(4 * time.Second)

	if err := relay.processEvent(context.Background(), event); err != nil {
		t.Fatalf("processEvent() error = %v", err)
	}
	if len(store.released) != 1 || store.released[0] != event.ID {
		t.Fatalf("released events = %v, want %s", store.released, event.ID)
	}
	if len(store.quarantined) != 0 {
		t.Fatalf("quarantined events = %v, want none", store.quarantined)
	}
	if store.releaseTime.Before(before) {
		t.Fatalf("next attempt = %v, want at least %v", store.releaseTime, before)
	}
	if store.lastError != "broker unavailable" {
		t.Fatalf("last error = %q", store.lastError)
	}
}

func TestRelayQuarantinesPermanentPublicationFailure(t *testing.T) {
	t.Parallel()

	event := Event{ID: uuid.New(), AggregateID: uuid.New()}
	store := &relayStoreStub{claim: func(context.Context) ([]Event, error) { return nil, nil }}
	relay := NewRelay(store, relayPublisherStub{err: permanentTestError{errors.New("message too large")}}, Config{})

	if err := relay.processEvent(context.Background(), event); err != nil {
		t.Fatalf("processEvent() error = %v", err)
	}
	if len(store.quarantined) != 1 || store.quarantined[0] != event.ID {
		t.Fatalf("quarantined events = %v, want %s", store.quarantined, event.ID)
	}
	if len(store.released) != 0 {
		t.Fatalf("released events = %v, want none", store.released)
	}
	if store.quarantineCode != "invalid_message" {
		t.Fatalf("quarantine code = %q, want invalid_message", store.quarantineCode)
	}
}

func TestRelayQuarantinesUnknownFailureAfterBudget(t *testing.T) {
	t.Parallel()

	event := Event{ID: uuid.New(), AggregateID: uuid.New(), PublishFailures: 2}
	store := &relayStoreStub{claim: func(context.Context) ([]Event, error) { return nil, nil }}
	relay := NewRelay(store, relayPublisherStub{err: errors.New("mystery failure")}, Config{MaxUnknownPublishFailures: 3})

	if err := relay.processEvent(context.Background(), event); err != nil {
		t.Fatalf("processEvent() error = %v", err)
	}
	if len(store.quarantined) != 1 || store.quarantined[0] != event.ID {
		t.Fatalf("quarantined events = %v, want %s", store.quarantined, event.ID)
	}
}

func TestRelayContinuesBatchAfterQuarantiningPoisonEvent(t *testing.T) {
	t.Parallel()

	poison := Event{ID: uuid.New(), AggregateID: uuid.New()}
	healthy := Event{ID: uuid.New(), AggregateID: uuid.New()}
	store := &relayStoreStub{claim: func(context.Context) ([]Event, error) { return []Event{poison, healthy}, nil }}
	publisher := &sequencePublisher{errors: []error{permanentTestError{errors.New("invalid message")}, nil}}
	relay := NewRelay(store, publisher, Config{})

	processed, err := relay.processBatch(context.Background())
	if err != nil {
		t.Fatalf("processBatch() error = %v", err)
	}
	if processed != 2 {
		t.Fatalf("processed = %d, want 2", processed)
	}
	if len(store.quarantined) != 1 || store.quarantined[0] != poison.ID {
		t.Fatalf("quarantined events = %v, want %s", store.quarantined, poison.ID)
	}
	if len(store.marked) != 1 || store.marked[0] != healthy.ID {
		t.Fatalf("marked events = %v, want %s", store.marked, healthy.ID)
	}
}

type sequencePublisher struct {
	errors []error
	calls  int
}

func (publisher *sequencePublisher) Publish(context.Context, string, []byte, map[string]string, string) error {
	err := publisher.errors[publisher.calls]
	publisher.calls++
	return err
}

func TestFailureBackoffIsBounded(t *testing.T) {
	t.Parallel()

	for attempt, want := range []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 5 * time.Second, 5 * time.Second} {
		if got := failureBackoff(attempt+1, time.Second, 5*time.Second); got != want {
			t.Fatalf("failureBackoff(%d) = %v, want %v", attempt+1, got, want)
		}
	}
}

func TestRetryBackoffCapsAtFifteenMinutes(t *testing.T) {
	t.Parallel()

	if got := retryBackoff(11); got != 15*time.Minute {
		t.Fatalf("retryBackoff(11) = %v, want %v", got, 15*time.Minute)
	}
}
