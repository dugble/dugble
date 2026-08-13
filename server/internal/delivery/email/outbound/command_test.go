package emaildelivery

import (
	"testing"

	"github.com/google/uuid"
)

func TestDeliveryEventsRouteByEmailStream(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		stream    string
		subject   string
		eventType string
	}{
		{
			name:      "transactional",
			stream:    "transactional",
			subject:   DeliverSubject,
			eventType: "email.transactional.send.requested.v1",
		},
		{
			name:      "marketing",
			stream:    "marketing",
			subject:   MarketingDeliverSubject,
			eventType: "email.marketing.send.requested.v1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			event, err := newDeliveryEvent(uuid.New(), uuid.New(), test.stream)
			if err != nil {
				t.Fatalf("newDeliveryEvent() error = %v", err)
			}
			if event.Subject != test.subject {
				t.Fatalf("subject = %q, want %q", event.Subject, test.subject)
			}
			if event.Headers["Dugble-Event-Type"] != test.eventType {
				t.Fatalf("event type = %q, want %q", event.Headers["Dugble-Event-Type"], test.eventType)
			}
		})
	}
}

func TestConsumerRoutesHaveIndependentDurables(t *testing.T) {
	t.Parallel()

	transactionalName, transactionalSubject := consumerRoute("transactional")
	marketingName, marketingSubject := consumerRoute("marketing")
	if transactionalName == marketingName {
		t.Fatalf("consumer names must differ: %q", transactionalName)
	}
	if transactionalSubject == marketingSubject {
		t.Fatalf("consumer subjects must differ: %q", transactionalSubject)
	}
}
