package email

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSendResponsesUseEmailObjectEnvelope(t *testing.T) {
	t.Parallel()

	single := (Message{ID: "email-1"}).SendResponse()
	if single.Object != "email" || single.ID != "email-1" {
		t.Fatalf("SendResponse() = %+v", single)
	}

	batch := SendResponses([]Message{{ID: "email-1"}, {ID: "email-2"}})
	if len(batch) != 2 || batch[0].Object != "email" || batch[0].ID != "email-1" || batch[1].Object != "email" || batch[1].ID != "email-2" {
		t.Fatalf("SendResponses() = %+v", batch)
	}
}

func TestRetrieveResponseOmitsInternalMessageType(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal((Message{ID: "email-1", MessageType: MessageTypeTransactional}).RetrieveResponse())
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if strings.Contains(string(encoded), `"stream"`) {
		t.Fatalf("RetrieveResponse() exposed internal stream: %s", encoded)
	}
}

func TestEventListResponseUsesPublicLifecycleContract(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(EventListResponse{Object: "list", Data: []Event{{ID: "event-1", Type: "delivered"}}})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if !strings.Contains(string(encoded), `"object":"list"`) || !strings.Contains(string(encoded), `"type":"delivered"`) {
		t.Fatalf("event list response = %s", encoded)
	}
}

func TestEmailLifecycleEventIncludesStableFailureCode(t *testing.T) {
	t.Parallel()
	code := "EMAIL_REJECTED"
	encoded, err := json.Marshal(Event{ID: "event-1", Type: "rejected", Code: &code})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if !strings.Contains(string(encoded), `"code":"EMAIL_REJECTED"`) {
		t.Fatalf("event = %s", encoded)
	}
}
