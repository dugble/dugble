package resend_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dugble/relay/email"
	"github.com/dugble/relay/providers/resend"
)

func TestSendAccepted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/emails" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer re_test" {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}
		var payload struct {
			From    string   `json:"from"`
			To      []string `json:"to"`
			Subject string   `json:"subject"`
			Text    string   `json:"text"`
			HTML    string   `json:"html"`
			ReplyTo string   `json:"reply_to"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.From != `"Acme" <sender@example.com>` {
			t.Fatalf("from = %q", payload.From)
		}
		if len(payload.To) != 1 || payload.To[0] != `"Customer" <customer@example.com>` {
			t.Fatalf("to = %#v", payload.To)
		}
		if payload.Subject != "Receipt" || payload.Text != "thanks" || payload.HTML != "<p>thanks</p>" {
			t.Fatalf("payload = %+v", payload)
		}
		if payload.ReplyTo != `"Support" <support@example.com>` {
			t.Fatalf("reply_to = %q", payload.ReplyTo)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"email-123"}`))
	}))
	defer server.Close()

	provider, err := resend.New(resend.Config{APIKey: "re_test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), message())
	if err != nil {
		t.Fatal(err)
	}
	if result.State != email.SubmissionAccepted || result.ProviderMessageID != "email-123" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestSendTreatsOKWithMalformedBodyAsAccepted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{not-json`))
	}))
	defer server.Close()

	provider, err := resend.New(resend.Config{APIKey: "re_test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), message())
	if err != nil || result.State != email.SubmissionAccepted {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestSendClassifiesValidationFailureAsRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"name":"invalid_from_address","message":"Invalid from"}`))
	}))
	defer server.Close()

	provider, err := resend.New(resend.Config{APIKey: "re_test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), message())
	if err == nil || result.State != email.SubmissionRejected {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	var apiErr *resend.APIError
	if !errors.As(err, &apiErr) || apiErr.Type != "invalid_from_address" {
		t.Fatalf("error = %#v", err)
	}
}

func TestSendClassifiesRateLimitAsRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"name":"rate_limit_exceeded","message":"Too many requests"}`))
	}))
	defer server.Close()

	provider, err := resend.New(resend.Config{APIKey: "re_test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), message())
	if err == nil || result.State != email.SubmissionRejected {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestSendTreatsConcurrentIdempotentRequestAsUnknown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"name":"concurrent_idempotent_requests","message":"Request is in progress"}`))
	}))
	defer server.Close()

	provider, err := resend.New(resend.Config{APIKey: "re_test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), message())
	if err == nil || result.State != email.SubmissionUnknown {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestSendTreatsInvalidIdempotentRequestAsRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"name":"invalid_idempotent_request","message":"Payload differs"}`))
	}))
	defer server.Close()

	provider, err := resend.New(resend.Config{APIKey: "re_test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), message())
	if err == nil || result.State != email.SubmissionRejected {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestSendTreatsRequestTimeoutAndServerErrorsAsUnknown(t *testing.T) {
	for _, status := range []int{http.StatusRequestTimeout, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"name":"application_error","message":"try later"}`))
			}))
			defer server.Close()

			provider, err := resend.New(resend.Config{APIKey: "re_test", BaseURL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			result, err := provider.Send(context.Background(), message())
			if err == nil || result.State != email.SubmissionUnknown {
				t.Fatalf("result=%+v err=%v", result, err)
			}
		})
	}
}

func TestSendTreatsTransportFailureAsUnknown(t *testing.T) {
	provider, err := resend.New(resend.Config{
		APIKey: "re_test",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("network down")
		})},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), message())
	if err == nil || result.State != email.SubmissionUnknown {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestCapabilitiesAndConfig(t *testing.T) {
	if _, err := resend.New(resend.Config{}); !errors.Is(err, resend.ErrInvalidConfig) {
		t.Fatalf("New() error = %v, want ErrInvalidConfig", err)
	}
	provider, err := resend.New(resend.Config{APIKey: "re_test"})
	if err != nil {
		t.Fatal(err)
	}
	capabilities := provider.Capabilities()
	if !capabilities.HTML || !capabilities.ReplyTo || !capabilities.MultipleRecipients || !capabilities.RequiresSubject || capabilities.MaxRecipients != 50 {
		t.Fatalf("capabilities = %+v", capabilities)
	}
}

func message() email.Message {
	return email.Message{
		From:    email.Address{Email: "sender@example.com", Name: "Acme"},
		To:      []email.Address{{Email: "customer@example.com", Name: "Customer"}},
		ReplyTo: &email.Address{Email: "support@example.com", Name: "Support"},
		Subject: "Receipt",
		Text:    "thanks",
		HTML:    "<p>thanks</p>",
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}
