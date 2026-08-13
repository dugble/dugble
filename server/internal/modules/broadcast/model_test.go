package broadcast

import (
	"encoding/json"
	"testing"
)

func TestExclusionSummaryJSONContract(t *testing.T) {
	t.Parallel()

	value := ExclusionSummary{
		Object:      "broadcast.exclusion_summary",
		BroadcastID: "broadcast-id",
		Total:       4,
		Reasons: map[string]int64{
			"global_unsubscribe": 2,
			"suppressed":         1,
			"topic_unsubscribed": 1,
		},
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded["object"] != "broadcast.exclusion_summary" || decoded["broadcast_id"] != "broadcast-id" || decoded["total"] != float64(4) {
		t.Fatalf("summary envelope = %s", encoded)
	}
	reasons, ok := decoded["reasons"].(map[string]any)
	if !ok || reasons["global_unsubscribe"] != float64(2) || reasons["suppressed"] != float64(1) || reasons["topic_unsubscribed"] != float64(1) {
		t.Fatalf("summary reasons = %s", encoded)
	}
}

func TestAnalyticsJSONContract(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(Analytics{
		Object: "broadcast.analytics", BroadcastID: "broadcast-id",
		Audience: 10, Delivered: 7, Bounced: 1, Opened: 4, Clicked: 2,
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded["object"] != "broadcast.analytics" || decoded["audience"] != float64(10) || decoded["delivered"] != float64(7) || decoded["opened"] != float64(4) {
		t.Fatalf("analytics response = %s", encoded)
	}
}

func TestUpdateRequestTopicIDJSONContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		body      string
		assertion func(*testing.T, UpdateRequest)
	}{
		{
			name: "omitted leaves topic unchanged",
			body: `{"revision":2}`,
			assertion: func(t *testing.T, request UpdateRequest) {
				t.Helper()
				if request.TopicID != nil {
					t.Fatalf("TopicID = %#v, want nil", request.TopicID)
				}
			},
		},
		{
			name: "null clears topic",
			body: `{"revision":2,"topic_id":null}`,
			assertion: func(t *testing.T, request UpdateRequest) {
				t.Helper()
				if request.TopicID == nil || *request.TopicID != nil {
					t.Fatalf("TopicID = %#v, want pointer to nil", request.TopicID)
				}
			},
		},
		{
			name: "string replaces topic",
			body: `{"revision":2,"topic_id":"0f593c7a-167e-4fe0-aeb8-6be39078d0f0"}`,
			assertion: func(t *testing.T, request UpdateRequest) {
				t.Helper()
				if request.TopicID == nil || *request.TopicID == nil || **request.TopicID != "0f593c7a-167e-4fe0-aeb8-6be39078d0f0" {
					t.Fatalf("TopicID = %#v, want replacement topic", request.TopicID)
				}
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var request UpdateRequest
			if err := json.Unmarshal([]byte(test.body), &request); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if request.Revision != 2 {
				t.Fatalf("Revision = %d, want 2", request.Revision)
			}
			test.assertion(t, request)
		})
	}
}
