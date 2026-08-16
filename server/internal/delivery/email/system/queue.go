package systememail

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dugble/dugble/server/internal/platform/outbox"
	platformemail "github.com/dugble/dugble/server/internal/providers/aws/ses"
)

type eventStore interface {
	Enqueue(context.Context, outbox.Event) (uuid.UUID, error)
	EnqueueTx(context.Context, pgx.Tx, outbox.Event) (uuid.UUID, error)
}

type Queue struct {
	store    eventStore
	defaults platformemail.DeliveryRoute
	region   string
	provider string
}

func NewQueue(store eventStore, defaults ...platformemail.Message) *Queue {
	queue := &Queue{store: store}
	if len(defaults) > 0 {
		queue.provider = strings.TrimSpace(defaults[0].Provider)
		queue.region = strings.TrimSpace(defaults[0].Region)
		queue.defaults = platformemail.DeliveryRoute{
			Stream:           strings.TrimSpace(defaults[0].Stream),
			ConfigurationSet: strings.TrimSpace(defaults[0].ConfigurationSet),
			SESTenantName:    strings.TrimSpace(defaults[0].SESTenantName),
		}
	}
	return queue
}

func (queue *Queue) Send(ctx context.Context, message platformemail.Message) (platformemail.Result, error) {
	return queue.send(ctx, nil, message)
}

// SendTx queues a system email in the caller's transaction so it is only
// published when the state change that caused it commits.
func (queue *Queue) SendTx(ctx context.Context, tx pgx.Tx, message platformemail.Message) (platformemail.Result, error) {
	if tx == nil {
		return platformemail.Result{}, ErrQueueNotConfigured
	}
	return queue.send(ctx, tx, message)
}

func (queue *Queue) send(ctx context.Context, tx pgx.Tx, message platformemail.Message) (platformemail.Result, error) {
	if queue == nil || queue.store == nil {
		return platformemail.Result{}, ErrQueueNotConfigured
	}
	if strings.TrimSpace(message.Provider) == "" {
		message.Provider = queue.provider
	}
	if strings.TrimSpace(message.Region) == "" {
		message.Region = queue.region
	}
	if strings.TrimSpace(message.Stream) == "" {
		message.Stream = queue.defaults.Stream
	}
	if strings.TrimSpace(message.ConfigurationSet) == "" {
		message.ConfigurationSet = queue.defaults.ConfigurationSet
	}
	if strings.TrimSpace(message.SESTenantName) == "" {
		message.SESTenantName = queue.defaults.SESTenantName
	}

	eventID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(deliveryNamespace+uuid.NewString()))
	payload, err := json.Marshal(DeliverCommand{EventID: eventID, Message: message, SchemaVersion: 1})
	if err != nil {
		return platformemail.Result{}, err
	}
	event := outbox.Event{
		ID:            eventID,
		Subject:       DeliverSubject,
		AggregateType: "system_email",
		AggregateID:   eventID,
		Payload:       payload,
		Headers: map[string]string{
			"Dugble-Event-Type": "email.system.send.requested.v1",
		},
	}
	if tx == nil {
		_, err = queue.store.Enqueue(ctx, event)
	} else {
		_, err = queue.store.EnqueueTx(ctx, tx, event)
	}
	if err != nil {
		return platformemail.Result{}, err
	}
	return platformemail.Result{Provider: "outbox", MessageID: eventID.String()}, nil
}
