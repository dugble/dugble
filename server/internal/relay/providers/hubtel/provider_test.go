package hubtel_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dugble/dugble/server/internal/relay/providers/hubtel"
	"github.com/dugble/dugble/server/internal/relay/sms"
)

func TestSendAccepted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/messages/send" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		username, password, ok := r.BasicAuth()
		if !ok || username != "client" || password != "secret" {
			t.Fatalf("unexpected basic auth: %q %q %v", username, password, ok)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["from"] != "Acme" || body["to"] != "+233200000000" || body["content"] != "hello" {
			t.Fatalf("unexpected body: %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"message":"success","responseCode":"0000","data":{"messageId":"msg-123","status":"Submitted"}}`))
	}))
	defer server.Close()

	provider, err := hubtel.New(hubtel.Config{
		ClientID:     "client",
		ClientSecret: "secret",
		BaseURL:      server.URL + "/v1/messages",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := provider.Send(context.Background(), sms.Message{
		To:   "+233200000000",
		From: "Acme",
		Text: "hello",
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if result.State != sms.SubmissionAccepted || result.ProviderMessageID != "msg-123" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestSendClassifiesClientErrorsAsRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"message":"rate limited"}`))
	}))
	defer server.Close()

	provider, err := hubtel.New(hubtel.Config{ClientID: "client", ClientSecret: "secret", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), sms.Message{To: "+233200000000", From: "Acme", Text: "hello"})
	if err == nil {
		t.Fatal("Send() error = nil, want HTTP error")
	}
	if result.State != sms.SubmissionRejected {
		t.Fatalf("state = %q, want rejected", result.State)
	}
}

func TestSendClassifiesServerErrorsAsUnknown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"internal error"}`))
	}))
	defer server.Close()

	provider, err := hubtel.New(hubtel.Config{ClientID: "client", ClientSecret: "secret", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), sms.Message{To: "+233200000000", From: "Acme", Text: "hello"})
	if err == nil {
		t.Fatal("Send() error = nil, want HTTP error")
	}
	if result.State != sms.SubmissionUnknown {
		t.Fatalf("state = %q, want unknown", result.State)
	}
}

func TestSendClassifiesTransportErrorsAsUnknown(t *testing.T) {
	client := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network timeout")
	})}
	provider, err := hubtel.New(hubtel.Config{
		ClientID:     "client",
		ClientSecret: "secret",
		BaseURL:      "https://example.invalid/v1/messages",
		HTTPClient:   client,
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := provider.Send(context.Background(), sms.Message{To: "+233200000000", From: "Acme", Text: "hello"})
	if err == nil {
		t.Fatal("Send() error = nil, want transport error")
	}
	if result.State != sms.SubmissionUnknown {
		t.Fatalf("state = %q, want unknown", result.State)
	}
}

func TestCapabilitiesMatchHubtelPreflightConstraints(t *testing.T) {
	provider, err := hubtel.New(hubtel.Config{ClientID: "client", ClientSecret: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	capabilities := provider.Capabilities()
	if !capabilities.AlphanumericSenderID || capabilities.MaxSenderIDLength != 11 || !capabilities.RequiresE164Recipient {
		t.Fatalf("unexpected capabilities: %+v", capabilities)
	}
}

func TestNewRequiresCredentials(t *testing.T) {
	_, err := hubtel.New(hubtel.Config{})
	if !errors.Is(err, hubtel.ErrInvalidConfig) {
		t.Fatalf("New() error = %v, want ErrInvalidConfig", err)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}
