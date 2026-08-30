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

func TestBroadcastJSONDoesNotExposeTemplateInternals(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(Broadcast{ID: "broadcast-id", Subject: "Hello", HTML: "<p>Hello</p>"})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if _, ok := decoded["template_id"]; ok {
		t.Fatalf("broadcast JSON exposes template_id: %s", encoded)
	}
	if _, ok := decoded["template_version_id"]; ok {
		t.Fatalf("broadcast JSON exposes template_version_id: %s", encoded)
	}
}

func TestUpdateRequestNullableFieldsJSONContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		body      string
		assertion func(*testing.T, UpdateRequest)
	}{
		{
			name: "omitted leaves nullable fields unchanged",
			body: `{"revision":2}`,
			assertion: func(t *testing.T, request UpdateRequest) {
				t.Helper()
				if request.TopicID != nil || request.FromName != nil || request.ReplyToEmail != nil || request.PreviewText != nil || request.Text != nil {
					t.Fatalf("nullable field unexpectedly present: %#v", request)
				}
			},
		},
		{
			name: "null clears nullable fields",
			body: `{"revision":2,"topic_id":null,"from_name":null,"reply_to_email":null,"preview_text":null,"text":null}`,
			assertion: func(t *testing.T, request UpdateRequest) {
				t.Helper()
				if request.TopicID == nil || *request.TopicID != nil {
					t.Fatalf("TopicID = %#v, want pointer to nil", request.TopicID)
				}
				if request.FromName == nil || *request.FromName != nil {
					t.Fatalf("FromName = %#v, want pointer to nil", request.FromName)
				}
				if request.ReplyToEmail == nil || *request.ReplyToEmail != nil {
					t.Fatalf("ReplyToEmail = %#v, want pointer to nil", request.ReplyToEmail)
				}
				if request.PreviewText == nil || *request.PreviewText != nil {
					t.Fatalf("PreviewText = %#v, want pointer to nil", request.PreviewText)
				}
				if request.Text == nil || *request.Text != nil {
					t.Fatalf("Text = %#v, want pointer to nil", request.Text)
				}
			},
		},
		{
			name: "strings replace nullable fields",
			body: `{"revision":2,"topic_id":"0f593c7a-167e-4fe0-aeb8-6be39078d0f0","from_name":"Dugble","reply_to_email":"reply@example.com","preview_text":"Preview","text":"Hello"}`,
			assertion: func(t *testing.T, request UpdateRequest) {
				t.Helper()
				if request.TopicID == nil || *request.TopicID == nil || **request.TopicID != "0f593c7a-167e-4fe0-aeb8-6be39078d0f0" {
					t.Fatalf("TopicID = %#v, want replacement topic", request.TopicID)
				}
				if request.FromName == nil || *request.FromName == nil || **request.FromName != "Dugble" {
					t.Fatalf("FromName = %#v, want Dugble", request.FromName)
				}
				if request.ReplyToEmail == nil || *request.ReplyToEmail == nil || **request.ReplyToEmail != "reply@example.com" {
					t.Fatalf("ReplyToEmail = %#v", request.ReplyToEmail)
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

func TestRenderPreviewUsesBroadcastContent(t *testing.T) {
	t.Parallel()

	preview, err := RenderBroadcast(Broadcast{
		FromEmail: "hello@example.com",
		Subject:   "Hello {{{NAME}}}",
		HTML:      "<p>Welcome {{{NAME}}}</p>",
		Text:      stringPtr("Welcome {{{NAME}}}"),
		VariableBindings: map[string]any{
			"NAME": "default",
		},
	}, map[string]any{"NAME": "Ada"})
	if err != nil {
		t.Fatalf("RenderBroadcast() error = %v", err)
	}

	if preview.Subject != "Hello Ada" {
		t.Fatalf("Subject = %q, want %q", preview.Subject, "Hello Ada")
	}
	if preview.HTML != "<p>Welcome Ada</p>" {
		t.Fatalf("HTML = %q", preview.HTML)
	}
	if preview.Text == nil || *preview.Text != "Welcome Ada" {
		t.Fatalf("Text = %#v", preview.Text)
	}
}

func TestRenderBroadcastRejectsMissingVariable(t *testing.T) {
	t.Parallel()

	_, err := RenderBroadcast(Broadcast{
		Subject: "Hello {{{NAME}}}",
		HTML:    "<p>Hello</p>",
	}, nil)
	if err == nil {
		t.Fatal("RenderBroadcast() error = nil, want missing variable error")
	}
}

func stringPtr(value string) *string { return &value }
