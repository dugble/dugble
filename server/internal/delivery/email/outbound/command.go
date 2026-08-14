package emaildelivery

import (
	"encoding/json"

	"github.com/google/uuid"

	"github.com/dugble/dugble/server/internal/platform/outbox"
)

const (
	DeliverSubject          = "dugble.job.email.send.v1"
	MarketingDeliverSubject = "dugble.job.email.send.marketing.v1"
	deliveryNamespace       = "https://dugble.com/events/email/send/"
)

type DeliverCommand struct {
	EventID       uuid.UUID `json:"event_id"`
	MessageID     uuid.UUID `json:"message_id"`
	TeamID        uuid.UUID `json:"team_id"`
	SchemaVersion int       `json:"schema_version"`
}

func newDeliveryEvent(messageID uuid.UUID, teamID uuid.UUID, stream string) (outbox.Event, error) {
	eventID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(deliveryNamespace+messageID.String()))
	payload, err := json.Marshal(DeliverCommand{
		EventID:       eventID,
		MessageID:     messageID,
		TeamID:        teamID,
		SchemaVersion: 1,
	})
	if err != nil {
		return outbox.Event{}, err
	}

	subject := DeliverSubject
	eventType := "email.transactional.send.requested.v1"
	if stream == "marketing" {
		subject = MarketingDeliverSubject
		eventType = "email.marketing.send.requested.v1"
	}
	return outbox.Event{
		ID:            eventID,
		Subject:       subject,
		AggregateType: "email_message",
		AggregateID:   messageID,
		Payload:       payload,
		Headers: map[string]string{
			"Dugble-Event-Type": eventType,
		},
	}, nil
}
