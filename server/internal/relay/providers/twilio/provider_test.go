package twilio_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/dugble/dugble/server/internal/relay/providers/twilio"
	"github.com/dugble/dugble/server/internal/relay/sms"
)

func TestSendAccepted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/2010-04-01/Accounts/AC123/Messages.json" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		username, password, ok := r.BasicAuth()
		if !ok || username != "SK123" || password != "secret" {
			t.Fatalf("unexpected basic auth: %q %q %v", username, password, ok)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		form, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatal(err)
		}
		if form.Get("To") != "+233200000000" || form.Get("From") != "Acme" || form.Get("Body") != "hello" {
			t.Fatalf("unexpected form: %#v", form)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"sid":"SM123","status":"queued"}`))
	}))
	defer server.Close()

	provider, err := twilio.New(twilio.Config{AccountSID: "AC123", APIKey: "SK123", APISecret: "secret", BaseURL: server.URL + "/2010-04-01"})
	if err != nil {
		t.Fatal(err)
	}

	result, err := provider.Send(context.Background(), sms.Message{To: "+233200000000", From: "Acme", Text: "hello"})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if result.State != sms.SubmissionAccepted || result.ProviderMessageID != "SM123" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestSendTreatsCreatedWithMalformedBodyAsAccepted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{not-json`))
	}))
	defer server.Close()
	provider, err := twilio.New(twilio.Config{AccountSID: "AC123", AuthToken: "token", BaseURL: server.URL})
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
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":21211,"message":"invalid to","status":400}`))
	}))
	defer server.Close()
	provider, err := twilio.New(twilio.Config{AccountSID: "AC123", AuthToken: "token", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), sms.Message{To: "+233200000000", From: "Acme", Text: "hello"})
	if err == nil || result.State != sms.SubmissionRejected {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	var apiErr *twilio.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != 21211 {
		t.Fatalf("error = %#v", err)
	}
}

func TestSendClassifiesRateLimitAsRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"code":20429,"message":"Too Many Requests","status":429}`))
	}))
	defer server.Close()
	provider, err := twilio.New(twilio.Config{AccountSID: "AC123", AuthToken: "token", BaseURL: server.URL})
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
		_, _ = w.Write([]byte(`{"message":"request timed out","status":408}`))
	}))
	defer server.Close()
	provider, err := twilio.New(twilio.Config{AccountSID: "AC123", AuthToken: "token", BaseURL: server.URL})
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
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"message":"unavailable","status":503}`))
	}))
	defer server.Close()
	provider, err := twilio.New(twilio.Config{AccountSID: "AC123", AuthToken: "token", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), sms.Message{To: "+233200000000", From: "Acme", Text: "hello"})
	if err == nil || result.State != sms.SubmissionUnknown {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestSendClassifiesTransportErrorsAsUnknown(t *testing.T) {
	client := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("timeout") })}
	provider, err := twilio.New(twilio.Config{AccountSID: "AC123", AuthToken: "token", HTTPClient: client})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), sms.Message{To: "+233200000000", From: "Acme", Text: "hello"})
	if err == nil || result.State != sms.SubmissionUnknown {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestCapabilitiesAllowNumericAndAlphanumericSenders(t *testing.T) {
	provider, err := twilio.New(twilio.Config{AccountSID: "AC123", AuthToken: "token"})
	if err != nil {
		t.Fatal(err)
	}
	capabilities := provider.Capabilities()
	if !capabilities.AlphanumericSenderID || !capabilities.RequiresE164Recipient {
		t.Fatalf("unexpected capabilities: %+v", capabilities)
	}
	if !capabilities.Supports(sms.Message{To: "+233200000000", From: "+14155552344"}) {
		t.Fatal("numeric E.164 sender should not be rejected by generic sender-length preflight")
	}
}

func TestNewRequiresCredentials(t *testing.T) {
	_, err := twilio.New(twilio.Config{AccountSID: "AC123"})
	if !errors.Is(err, twilio.ErrInvalidConfig) {
		t.Fatalf("New() error = %v, want ErrInvalidConfig", err)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)
func (fn roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) { return fn(request) }
