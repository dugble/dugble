package renewal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dugble/dugble/server/internal/platform/outbox"
)

const subscriptionEventNamespace = "dugble:billing:subscription:"

type eventStore interface {
	EnqueueTx(context.Context, pgx.Tx, outbox.Event) (uuid.UUID, error)
}

type EventPublisher struct {
	store eventStore
}

func NewEventPublisher(store eventStore) *EventPublisher {
	return &EventPublisher{store: store}
}

func (publisher *EventPublisher) PublishTx(ctx context.Context, tx pgx.Tx, result Result) error {
	if publisher == nil || publisher.store == nil {
		return errors.New("subscription renewal event publisher is not configured")
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("encode subscription renewal event: %w", err)
	}
	eventID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(fmt.Sprintf(
		"%s%s:%s:%s:%d",
		subscriptionEventNamespace,
		result.SubscriptionID,
		result.PeriodStart.UTC().Format("2006-01-02T15:04:05Z07:00"),
		result.Outcome,
		result.Charge.AttemptCount,
	)))
	_, err = publisher.store.EnqueueTx(ctx, tx, outbox.Event{
		ID:            eventID,
		Subject:       "billing.subscription." + string(result.Outcome),
		AggregateType: "subscription",
		AggregateID:   result.SubscriptionID,
		Payload:       payload,
		Headers: map[string]string{
			"event_type": "billing.subscription." + string(result.Outcome),
			"team_id":    result.TeamID.String(),
		},
	})
	if err != nil {
		return fmt.Errorf("enqueue subscription renewal event: %w", err)
	}
	return nil
}
