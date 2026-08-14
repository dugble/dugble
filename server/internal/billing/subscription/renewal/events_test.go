package renewal

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dugble/dugble/server/internal/platform/outbox"
)

type eventStoreStub struct {
	event outbox.Event
}

func (store *eventStoreStub) EnqueueTx(_ context.Context, _ pgx.Tx, event outbox.Event) (uuid.UUID, error) {
	store.event = event
	return event.ID, nil
}

func TestEventPublisherCreatesDeterministicSubscriptionEvent(t *testing.T) {
	store := &eventStoreStub{}
	publisher := NewEventPublisher(store)
	result := Result{
		SubscriptionID: uuid.New(), TeamID: uuid.New(), Outcome: OutcomeRenewed,
		PeriodStart: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
	}

	if err := publisher.PublishTx(context.Background(), nil, result); err != nil {
		t.Fatal(err)
	}
	firstID := store.event.ID
	if store.event.Subject != "billing.subscription.renewed" || store.event.AggregateID != result.SubscriptionID {
		t.Fatalf("event = %+v", store.event)
	}
	if err := publisher.PublishTx(context.Background(), nil, result); err != nil {
		t.Fatal(err)
	}
	if store.event.ID != firstID {
		t.Fatalf("event id = %s, want %s", store.event.ID, firstID)
	}
}
