package postmark_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dugble/relay/email"
	"github.com/dugble/relay/providers/postmark"
)

func TestSendAccepted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/email" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("X-Postmark-Server-Token") != "server-token" {
			t.Fatalf("server token = %q", r.Header.Get("X-Postmark-Server-Token"))
		}
		var payload struct {
			From          string `json:"From"`
			To            string `json:"To"`
			Subject       string `json:"Subject"`
			HTMLBody      string `json:"HtmlBody"`
			TextBody      string `json:"TextBody"`
			ReplyTo       string `json:"ReplyTo"`
			MessageStream string `json:"MessageStream"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.From != `"Acme" <sender@example.com>` {
			t.Fatalf("from = %q", payload.From)
		}
		if payload.To != `"Customer" <customer@example.com>,second@example.com` {
			t.Fatalf("to = %q", payload.To)
		}
		if payload.Subject != "Receipt" || payload.TextBody != "thanks" || payload.HTMLBody != "<p>thanks</p>" {
			t.Fatalf("payload = %+v", payload)
		}
		if payload.ReplyTo != `"Support" <support@example.com>` {
			t.Fatalf("reply_to = %q", payload.ReplyTo)
		}
		if payload.MessageStream != "transactional" {
			t.Fatalf("message stream = %q", payload.MessageStream)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ErrorCode":0,"Message":"OK","MessageID":"message-123"}`))
	}))
	defer server.Close()

	provider, err := postmark.New(postmark.Config{
		ServerToken:   "server-token",
		MessageStream: "transactional",
		BaseURL:       server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), message())
	if err != nil {
		t.Fatal(err)
	}
	if result.State != email.SubmissionAccepted || result.ProviderMessageID != "message-123" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestDefaultMessageStreamIsOutbound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			MessageStream string `json:"MessageStream"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.MessageStream != "outbound" {
			t.Fatalf("message stream = %q, want outbound", payload.MessageStream)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"MessageID":"message-123"}`))
	}))
	defer server.Close()

	provider, err := postmark.New(postmark.Config{ServerToken: "server-token", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), message())
	if err != nil || result.State != email.SubmissionAccepted {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestSendTreatsOKWithMalformedBodyAsAccepted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{not-json`))
	}))
	defer server.Close()

	provider, err := postmark.New(postmark.Config{ServerToken: "server-token", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), message())
	if err != nil || result.State != email.SubmissionAccepted {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestSendClassifiesDocumentedClientErrorsAsRejected(t *testing.T) {
	for _, status := range []int{
		http.StatusUnauthorized,
		http.StatusNotFound,
		http.StatusRequestEntityTooLarge,
		http.StatusUnsupportedMediaType,
		http.StatusUnprocessableEntity,
		http.StatusTooManyRequests,
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"ErrorCode":300,"Message":"request rejected"}`))
			}))
			defer server.Close()

			provider, err := postmark.New(postmark.Config{ServerToken: "server-token", BaseURL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			result, err := provider.Send(context.Background(), message())
			if err == nil || result.State != email.SubmissionRejected {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			var apiErr *postmark.APIError
			if !errors.As(err, &apiErr) || apiErr.ErrorCode != 300 || apiErr.StatusCode != status {
				t.Fatalf("error = %#v", err)
			}
		})
	}
}

func TestSendTreatsAmbiguousStatusesAsUnknown(t *testing.T) {
	for _, status := range []int{http.StatusRequestTimeout, http.StatusInternalServerError, http.StatusServiceUnavailable, http.StatusConflict} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"ErrorCode":500,"Message":"try later"}`))
			}))
			defer server.Close()

			provider, err := postmark.New(postmark.Config{ServerToken: "server-token", BaseURL: server.URL})
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
	provider, err := postmark.New(postmark.Config{
		ServerToken: "server-token",
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
	if _, err := postmark.New(postmark.Config{}); !errors.Is(err, postmark.ErrInvalidConfig) {
		t.Fatalf("New() error = %v, want ErrInvalidConfig", err)
	}
	provider, err := postmark.New(postmark.Config{ServerToken: "server-token"})
	if err != nil {
		t.Fatal(err)
	}
	capabilities := provider.Capabilities()
	if !capabilities.HTML || !capabilities.ReplyTo || !capabilities.MultipleRecipients || capabilities.MaxRecipients != 50 || capabilities.RequiresSubject {
		t.Fatalf("capabilities = %+v", capabilities)
	}
}

func message() email.Message {
	return email.Message{
		From: email.Address{Email: "sender@example.com", Name: "Acme"},
		To: []email.Address{
			{Email: "customer@example.com", Name: "Customer"},
			{Email: "second@example.com"},
		},
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
