package vonage_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dugble/relay/providers/vonage"
	"github.com/dugble/relay/sms"
)

func TestSendAccepted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/messages" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		username, password, ok := r.BasicAuth()
		if !ok || username != "api-key" || password != "secret" {
			t.Fatalf("unexpected basic auth: %q %q %v", username, password, ok)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["channel"] != "sms" || body["message_type"] != "text" || body["to"] != "233200000000" || body["from"] != "233240000000" || body["text"] != "hello" {
			t.Fatalf("unexpected body: %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"message_uuid":"aaaaaaaa-bbbb-4ccc-8ddd-0123456789ab"}`))
	}))
	defer server.Close()

	provider, err := vonage.New(vonage.Config{APIKey: "api-key", APISecret: "secret", BaseURL: server.URL + "/v1/messages"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), sms.Message{To: "+233200000000", From: "+233240000000", Text: "hello"})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if result.State != sms.SubmissionAccepted || result.ProviderMessageID != "aaaaaaaa-bbbb-4ccc-8ddd-0123456789ab" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestSendPreservesAlphanumericSender(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["from"] != "Acme" {
			t.Fatalf("from = %q", body["from"])
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"message_uuid":"msg-123"}`))
	}))
	defer server.Close()

	provider, err := vonage.New(vonage.Config{APIKey: "api-key", APISecret: "secret", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), sms.Message{To: "+233200000000", From: "Acme", Text: "hello"})
	if err != nil || result.State != sms.SubmissionAccepted {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestSendTreatsAcceptedWithMalformedBodyAsAccepted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{not-json`))
	}))
	defer server.Close()

	provider, err := vonage.New(vonage.Config{APIKey: "api-key", APISecret: "secret", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), sms.Message{To: "+233200000000", From: "Acme", Text: "hello"})
	if err != nil || result.State != sms.SubmissionAccepted {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestSendClassifiesClientErrorsAsRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"type":"https://developer.vonage.com/api-errors#invalid-request","title":"Invalid Request","detail":"invalid to"}`))
	}))
	defer server.Close()

	provider, err := vonage.New(vonage.Config{APIKey: "api-key", APISecret: "secret", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), sms.Message{To: "+233200000000", From: "Acme", Text: "hello"})
	if err == nil || result.State != sms.SubmissionRejected {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	var apiErr *vonage.APIError
	if !errors.As(err, &apiErr) || apiErr.Detail != "invalid to" {
		t.Fatalf("error = %#v", err)
	}
}

func TestSendClassifiesRateLimitAsRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"type":"https://developer.vonage.com/api-errors#throttled","title":"Too Many Requests","detail":"rate limited"}`))
	}))
	defer server.Close()

	provider, err := vonage.New(vonage.Config{APIKey: "api-key", APISecret: "secret", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), sms.Message{To: "+233200000000", From: "Acme", Text: "hello"})
	if err == nil || result.State != sms.SubmissionRejected {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestSendTreatsRequestTimeoutAsUnknown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusRequestTimeout)
		_, _ = w.Write([]byte(`{"title":"Request Timeout","detail":"request timed out"}`))
	}))
	defer server.Close()

	provider, err := vonage.New(vonage.Config{APIKey: "api-key", APISecret: "secret", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), sms.Message{To: "+233200000000", From: "Acme", Text: "hello"})
	if err == nil || result.State != sms.SubmissionUnknown {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestSendClassifiesServerErrorsAsUnknown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"title":"Internal Error","detail":"try later"}`))
	}))
	defer server.Close()

	provider, err := vonage.New(vonage.Config{APIKey: "api-key", APISecret: "secret", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), sms.Message{To: "+233200000000", From: "Acme", Text: "hello"})
	if err == nil || result.State != sms.SubmissionUnknown {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestSendClassifiesTransportErrorsAsUnknown(t *testing.T) {
	client := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("timeout")
	})}
	provider, err := vonage.New(vonage.Config{APIKey: "api-key", APISecret: "secret", HTTPClient: client})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), sms.Message{To: "+233200000000", From: "Acme", Text: "hello"})
	if err == nil || result.State != sms.SubmissionUnknown {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestCapabilitiesRequireE164Recipient(t *testing.T) {
	provider, err := vonage.New(vonage.Config{APIKey: "api-key", APISecret: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	capabilities := provider.Capabilities()
	if !capabilities.AlphanumericSenderID || !capabilities.RequiresE164Recipient {
		t.Fatalf("unexpected capabilities: %+v", capabilities)
	}
}

func TestNewRequiresCredentials(t *testing.T) {
	_, err := vonage.New(vonage.Config{})
	if !errors.Is(err, vonage.ErrInvalidConfig) {
		t.Fatalf("New() error = %v, want ErrInvalidConfig", err)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}
