package broadcast

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
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

func TestBroadcastJSONOwnsMessageContent(t *testing.T) {
	t.Parallel()

	fromName := "Dugble"
	replyTo := "support@example.com"
	previewText := "A quick preview"
	textBody := "Hello {{{FIRST_NAME}}}"
	encoded, err := json.Marshal(Broadcast{
		ID:           "broadcast-id",
		TeamID:       "team-id",
		Name:         "Product update",
		Status:       StatusDraft,
		SegmentID:    "segment-id",
		FromEmail:    "hello@example.com",
		FromName:     &fromName,
		ReplyToEmail: &replyTo,
		Subject:      "Hello {{{FIRST_NAME}}}",
		PreviewText:  &previewText,
		HTML:         "<p>Hello {{{FIRST_NAME}}}</p>",
		Text:         &textBody,
		VariableBindings: map[string]any{
			"FIRST_NAME": "there",
		},
		Revision: 1,
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	for _, key := range []string{
		"from_email", "from_name", "reply_to_email", "subject", "preview_text", "html", "text", "variable_bindings",
	} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("broadcast JSON missing %q: %s", key, encoded)
		}
	}
	for _, key := range []string{
		"template", "template_id", "template_version_id", "source_template_id", "source_template_version_id",
	} {
		if _, ok := decoded[key]; ok {
			t.Fatalf("broadcast JSON exposes %q: %s", key, encoded)
		}
	}
}

func TestCreateRequestJSONContract(t *testing.T) {
	t.Parallel()

	const body = `{
		"name":"Product update",
		"segment_id":"0f593c7a-167e-4fe0-aeb8-6be39078d0f0",
		"topic_id":"381a82e7-e37e-42de-b902-23e34cf89d42",
		"from_email":"hello@example.com",
		"from_name":"Dugble",
		"reply_to_email":"support@example.com",
		"subject":"Hello {{{FIRST_NAME}}}",
		"preview_text":"Welcome {{{FIRST_NAME}}}",
		"html":"<p>Hello {{{FIRST_NAME}}}</p>",
		"text":"Hello {{{FIRST_NAME}}}",
		"variable_bindings":{"FIRST_NAME":"there"},
		"send":true,
		"scheduled_at":"2030-09-01T09:00:00Z"
	}`

	var request CreateRequest
	if err := json.Unmarshal([]byte(body), &request); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if request.Name != "Product update" || request.Subject != "Hello {{{FIRST_NAME}}}" || request.HTML != "<p>Hello {{{FIRST_NAME}}}</p>" {
		t.Fatalf("message content = %#v", request)
	}
	if request.FromEmail == nil || *request.FromEmail != "hello@example.com" {
		t.Fatalf("FromEmail = %#v", request.FromEmail)
	}
	if request.ReplyToEmail == nil || *request.ReplyToEmail != "support@example.com" {
		t.Fatalf("ReplyToEmail = %#v", request.ReplyToEmail)
	}
	if request.VariableBindings["FIRST_NAME"] != "there" {
		t.Fatalf("VariableBindings = %#v", request.VariableBindings)
	}
	if !request.Send || request.ScheduledAt == nil || request.ScheduledAt.Format(time.RFC3339) != "2030-09-01T09:00:00Z" {
		t.Fatalf("send scheduling = send:%v scheduled_at:%v", request.Send, request.ScheduledAt)
	}
}

func TestSendRequestJSONContract(t *testing.T) {
	t.Parallel()

	var empty SendRequest
	if err := json.Unmarshal([]byte(`{}`), &empty); err != nil {
		t.Fatalf("Unmarshal(empty) error = %v", err)
	}
	if empty.ScheduledAt != nil {
		t.Fatalf("empty ScheduledAt = %v, want nil", empty.ScheduledAt)
	}

	var scheduled SendRequest
	if err := json.Unmarshal([]byte(`{"scheduled_at":"2030-09-01T09:00:00Z"}`), &scheduled); err != nil {
		t.Fatalf("Unmarshal(scheduled) error = %v", err)
	}
	if scheduled.ScheduledAt == nil || scheduled.ScheduledAt.Format(time.RFC3339) != "2030-09-01T09:00:00Z" {
		t.Fatalf("ScheduledAt = %v", scheduled.ScheduledAt)
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
				if request.TopicID != nil || request.FromEmail != nil || request.FromName != nil || request.ReplyToEmail != nil || request.PreviewText != nil || request.Text != nil {
					t.Fatalf("nullable field unexpectedly present: %#v", request)
				}
			},
		},
		{
			name: "null clears nullable fields",
			body: `{"revision":2,"topic_id":null,"from_email":null,"from_name":null,"reply_to_email":null,"preview_text":null,"text":null}`,
			assertion: func(t *testing.T, request UpdateRequest) {
				t.Helper()
				assertNullableCleared(t, "TopicID", request.TopicID)
				assertNullableCleared(t, "FromEmail", request.FromEmail)
				assertNullableCleared(t, "FromName", request.FromName)
				assertNullableCleared(t, "ReplyToEmail", request.ReplyToEmail)
				assertNullableCleared(t, "PreviewText", request.PreviewText)
				assertNullableCleared(t, "Text", request.Text)
			},
		},
		{
			name: "strings replace nullable fields",
			body: `{"revision":2,"topic_id":"0f593c7a-167e-4fe0-aeb8-6be39078d0f0","from_email":"hello@example.com","from_name":"Dugble","reply_to_email":"reply@example.com","preview_text":"Preview","text":"Hello"}`,
			assertion: func(t *testing.T, request UpdateRequest) {
				t.Helper()
				assertNullableValue(t, "TopicID", request.TopicID, "0f593c7a-167e-4fe0-aeb8-6be39078d0f0")
				assertNullableValue(t, "FromEmail", request.FromEmail, "hello@example.com")
				assertNullableValue(t, "FromName", request.FromName, "Dugble")
				assertNullableValue(t, "ReplyToEmail", request.ReplyToEmail, "reply@example.com")
				assertNullableValue(t, "PreviewText", request.PreviewText, "Preview")
				assertNullableValue(t, "Text", request.Text, "Hello")
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

func TestPreviewResponseJSONContract(t *testing.T) {
	t.Parallel()

	previewText := "Welcome Ada"
	textBody := "Hello Ada"
	encoded, err := json.Marshal(PreviewResponse{
		FromEmail:   "hello@example.com",
		Subject:     "Hello Ada",
		PreviewText: &previewText,
		HTML:        "<p>Hello Ada</p>",
		Text:        &textBody,
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded["from_email"] != "hello@example.com" || decoded["subject"] != "Hello Ada" || decoded["preview_text"] != "Welcome Ada" || decoded["html"] != "<p>Hello Ada</p>" || decoded["text"] != "Hello Ada" {
		t.Fatalf("preview response = %s", encoded)
	}
	if _, ok := decoded["template_id"]; ok {
		t.Fatalf("preview response exposes template_id: %s", encoded)
	}
	if _, ok := decoded["version_id"]; ok {
		t.Fatalf("preview response exposes version_id: %s", encoded)
	}
}

func TestRenderBroadcastVariablePrecedenceAndEscaping(t *testing.T) {
	t.Parallel()

	preview, err := RenderBroadcast(Broadcast{
		FromEmail:   "hello@example.com",
		FromName:    stringPtr("Dugble"),
		Subject:     "Hello {{{NAME}}}",
		PreviewText: stringPtr("Preview {{{NAME}}}"),
		HTML:        "<p>Welcome {{{NAME}}}</p>",
		Text:        stringPtr("Welcome {{{NAME}}}"),
		VariableBindings: map[string]any{
			"NAME": "default",
		},
	}, map[string]any{"NAME": "<Ada>"})
	if err != nil {
		t.Fatalf("RenderBroadcast() error = %v", err)
	}

	if preview.Subject != "Hello <Ada>" {
		t.Fatalf("Subject = %q", preview.Subject)
	}
	if preview.PreviewText == nil || *preview.PreviewText != "Preview <Ada>" {
		t.Fatalf("PreviewText = %#v", preview.PreviewText)
	}
	if preview.HTML != "<p>Welcome &lt;Ada&gt;</p>" {
		t.Fatalf("HTML = %q", preview.HTML)
	}
	if preview.Text == nil || *preview.Text != "Welcome <Ada>" {
		t.Fatalf("Text = %#v", preview.Text)
	}
}

func TestRenderFanoutRecipientUsesOwnedContent(t *testing.T) {
	t.Parallel()

	fromEmail := "marketing@example.com"
	preview, err := RenderFanoutRecipient(FanoutRecipient{
		ID:          uuid.New(),
		TeamID:      uuid.New(),
		BroadcastID: uuid.New(),
		Email:       "ada@example.com",
		FromEmail:   &fromEmail,
		Subject:     "Hello {{{FIRST_NAME}}}",
		HTML:        "<p>Hello {{{FIRST_NAME}}}</p>",
		VariableBindings: map[string]any{
			"FIRST_NAME": "there",
		},
	}, map[string]any{"FIRST_NAME": "Ada"})
	if err != nil {
		t.Fatalf("RenderFanoutRecipient() error = %v", err)
	}
	if preview.FromEmail != fromEmail || preview.Subject != "Hello Ada" || preview.HTML != "<p>Hello Ada</p>" {
		t.Fatalf("fanout preview = %#v", preview)
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

func assertNullableCleared(t *testing.T, name string, value **string) {
	t.Helper()
	if value == nil || *value != nil {
		t.Fatalf("%s = %#v, want pointer to nil", name, value)
	}
}

func assertNullableValue(t *testing.T, name string, value **string, want string) {
	t.Helper()
	if value == nil || *value == nil || **value != want {
		t.Fatalf("%s = %#v, want %q", name, value, want)
	}
}

func stringPtr(value string) *string { return &value }
