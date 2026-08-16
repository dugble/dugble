package mnotify_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dugble/relay/providers/mnotify"
	"github.com/dugble/relay/sms"
)

func TestSendAccepted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/sms/quick" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("key") != "secret" {
			t.Fatalf("API key = %q", r.URL.Query().Get("key"))
		}
		var body struct {
			Recipient  []string `json:"recipient"`
			Sender     string   `json:"sender"`
			Message    string   `json:"message"`
			IsSchedule bool     `json:"is_schedule"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.Recipient) != 1 || body.Recipient[0] != "233200000000" || body.Sender != "Acme" || body.Message != "hello" || body.IsSchedule {
			t.Fatalf("unexpected body: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","code":2000,"message":"messages sent successfully","summary":{"message_id":"msg-123"}}`))
	}))
	defer server.Close()

	provider, err := mnotify.New(mnotify.Config{APIKey: "secret", BaseURL: server.URL + "/api/sms/quick"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), sms.Message{To: "+233200000000", From: "Acme", Text: "hello"})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if result.State != sms.SubmissionAccepted || result.ProviderMessageID != "msg-123" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestSendTreatsDocumentedSuccessCodeAsAcceptedWhenString(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"success","code":"2000","message":"messages sent successfully"}`))
	}))
	defer server.Close()
	provider, err := mnotify.New(mnotify.Config{APIKey: "secret", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), sms.Message{To: "0241234567", From: "Acme", Text: "hello"})
	if err != nil || result.State != sms.SubmissionAccepted {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestSendTreatsErrorResponseAsUnknown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status":"error","code":4000,"message":"invalid request"}`))
	}))
	defer server.Close()
	provider, err := mnotify.New(mnotify.Config{APIKey: "secret", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), sms.Message{To: "0241234567", From: "Acme", Text: "hello"})
	if err == nil {
		t.Fatal("Send() error = nil, want API error")
	}
	if result.State != sms.SubmissionUnknown {
		t.Fatalf("state = %q, want unknown", result.State)
	}
}

func TestSendTreatsTransportErrorAsUnknown(t *testing.T) {
	client := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("timeout")
	})}
	provider, err := mnotify.New(mnotify.Config{APIKey: "secret", HTTPClient: client})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), sms.Message{To: "0241234567", From: "Acme", Text: "hello"})
	if err == nil || result.State != sms.SubmissionUnknown {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestNewRequiresAPIKey(t *testing.T) {
	_, err := mnotify.New(mnotify.Config{})
	if !errors.Is(err, mnotify.ErrInvalidConfig) {
		t.Fatalf("New() error = %v, want ErrInvalidConfig", err)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}
