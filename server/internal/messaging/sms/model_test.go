package sms

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSMSResponseOmitsRemovedTagsField(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal(Message{Metadata: json.RawMessage(`{}`)}.Response())
	if err != nil {
		t.Fatalf("marshal SMS response: %v", err)
	}

	var response map[string]json.RawMessage
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("decode SMS response: %v", err)
	}
	if _, exists := response["tags"]; exists {
		t.Fatalf("response contains removed tags field: %s", payload)
	}
}

func TestSMSLifecycleEventIncludesStableFailureCode(t *testing.T) {
	t.Parallel()
	code := "SMS_REJECTED"
	payload, err := json.Marshal(EventListResponse{Object: "list", Data: []Event{{ID: "event-1", Type: "rejected", Code: &code}}})
	if err != nil {
		t.Fatalf("marshal SMS events: %v", err)
	}
	if !strings.Contains(string(payload), `"code":"SMS_REJECTED"`) {
		t.Fatalf("events = %s", payload)
	}
}
