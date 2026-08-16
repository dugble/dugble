package moolre_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dugble/dugble/server/internal/relay/providers/moolre"
	"github.com/dugble/dugble/server/internal/relay/sms"
)

func TestSendAccepted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/open/sms/send" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("X-API-VASKEY"); got != "vas-secret" {
			t.Fatalf("X-API-VASKEY = %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q", got)
		}
		var payload struct {
			Type     int    `json:"type"`
			SenderID string `json:"senderid"`
			Messages []struct {
				Recipient string `json:"recipient"`
				Message   string `json:"message"`
				Ref       string `json:"ref"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.Type != 1 || payload.SenderID != "Acme" {
			t.Fatalf("payload = %+v", payload)
		}
		if len(payload.Messages) != 1 || payload.Messages[0].Recipient != "0241234567" || payload.Messages[0].Message != "hello" || payload.Messages[0].Ref != "" {
			t.Fatalf("messages = %+v", payload.Messages)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":1,"code":"SMS01","message":"Success","data":null,"go":null}`))
	}))
	defer server.Close()

	provider, err := moolre.New(moolre.Config{VASKey: "vas-secret", BaseURL: server.URL + "/open/sms/send"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), sms.Message{To: "0241234567", From: "Acme", Text: "hello"})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if result.State != sms.SubmissionAccepted {
		t.Fatalf("state = %q, want accepted", result.State)
	}
}

func TestSendClassifiesDocumentedRejectionsAsRejected(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		code   string
	}{
		{name: "unapproved sender", status: http.StatusBadRequest, body: `{"status":0,"code":"ASMS07","message":"Sender ID is not approved"}`, code: "ASMS07"},
		{name: "authentication", status: http.StatusUnauthorized, body: `{"status":0,"code":"AIN01","message":"Authentication Error"}`, code: "AIN01"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			provider, err := moolre.New(moolre.Config{VASKey: "vas-secret", BaseURL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			result, err := provider.Send(context.Background(), sms.Message{To: "0241234567", From: "Acme", Text: "hello"})
			if err == nil || result.State != sms.SubmissionRejected {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			var apiErr *moolre.APIError
			if !errors.As(err, &apiErr) || apiErr.Code != tc.code {
				t.Fatalf("error = %#v", err)
			}
		})
	}
}

func TestSendTreatsAmbiguousResponsesAsUnknown(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
	}{
		{name: "server error", status: http.StatusInternalServerError, body: `{"status":0,"code":"ERR","message":"try later"}`},
		{name: "unexpected client status", status: http.StatusForbidden, body: `{"status":0,"code":"DENIED","message":"denied"}`},
		{name: "non-success 200", status: http.StatusOK, body: `{"status":0,"code":"OTHER","message":"not accepted"}`},
		{name: "malformed success body", status: http.StatusOK, body: `{not-json`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			provider, err := moolre.New(moolre.Config{VASKey: "vas-secret", BaseURL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			result, err := provider.Send(context.Background(), sms.Message{To: "0241234567", From: "Acme", Text: "hello"})
			if err == nil || result.State != sms.SubmissionUnknown {
				t.Fatalf("result=%+v err=%v", result, err)
			}
		})
	}
}

func TestSendTreatsTransportFailureAsUnknown(t *testing.T) {
	provider, err := moolre.New(moolre.Config{
		VASKey: "vas-secret",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("network timeout")
		})},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Send(context.Background(), sms.Message{To: "0241234567", From: "Acme", Text: "hello"})
	if err == nil || result.State != sms.SubmissionUnknown {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestCapabilitiesAndConfig(t *testing.T) {
	if _, err := moolre.New(moolre.Config{}); !errors.Is(err, moolre.ErrInvalidConfig) {
		t.Fatalf("New() error = %v, want ErrInvalidConfig", err)
	}
	provider, err := moolre.New(moolre.Config{VASKey: "vas-secret"})
	if err != nil {
		t.Fatal(err)
	}
	capabilities := provider.Capabilities()
	if !capabilities.AlphanumericSenderID || capabilities.MaxSenderIDLength != 11 || capabilities.RequiresE164Recipient {
		t.Fatalf("capabilities = %+v", capabilities)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}
