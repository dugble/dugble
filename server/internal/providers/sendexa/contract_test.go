package sendexa_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	provider "github.com/dugble/dugble/server/internal/providers"
	"github.com/dugble/dugble/server/internal/providers/sendexa"
	"github.com/dugble/dugble/server/internal/relay/relaytest"
	"github.com/dugble/dugble/server/internal/relay/sms"
)

func TestProviderContract(t *testing.T) {
	relaytest.Contract[sms.Message]{
		Name:     "sendexa",
		Accepted: scenario(http.StatusOK, `{"success":true,"data":{"messageId":"MSG-1","status":"queued"}}`),
		Unknown:  scenario(http.StatusBadRequest, `{"success":false,"message":"invalid request","data":{"status":"failed"}}`),
		RunRelay: func(ctx context.Context, primary, fallback relaytest.Provider[sms.Message], message sms.Message) (sms.SendResult, error) {
			r, err := sms.NewRelay(primary, fallback)
			if err != nil {
				return sms.SendResult{}, err
			}
			return r.Send(ctx, message)
		},
	}.Run(t)
}

func TestSendUsesHTTPContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/sms/send" {
			t.Fatalf("path=%s, want /v1/sms/send", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Basic token" {
			t.Fatalf("authorization=%q", got)
		}

		var payload struct {
			To      string `json:"to"`
			From    string `json:"from"`
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.To != "233555539152" || payload.From != "Vidzro" || payload.Message != "hello" {
			t.Fatalf("payload=%+v", payload)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"messageId":"MSG-cd57528e-60f3-4508-a6ef-e73982c6c78e","status":"sent"}}`))
	}))
	defer server.Close()

	p, err := sendexa.New(sendexa.Config{Token: "token", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}

	result, err := p.Send(context.Background(), sms.Message{
		To:   "+233555539152",
		From: "Vidzro",
		Text: "hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.State != sms.SubmissionAccepted {
		t.Fatalf("state=%s, want accepted", result.State)
	}
	if result.ProviderMessageID != "MSG-cd57528e-60f3-4508-a6ef-e73982c6c78e" {
		t.Fatalf("provider message ID=%q", result.ProviderMessageID)
	}
}

func TestCheckSMSStatus(t *testing.T) {
	tests := []struct {
		name       string
		native     string
		normalized provider.SMSStatus
	}{
		{name: "queued", native: "queued", normalized: provider.SMSPending},
		{name: "sent", native: "sent", normalized: provider.SMSPending},
		{name: "delivered", native: "delivered", normalized: provider.SMSDelivered},
		{name: "failed", native: "failed", normalized: provider.SMSFailed},
		{name: "expired", native: "expired", normalized: provider.SMSFailed},
		{name: "unknown", native: "carrier_pending", normalized: provider.SMSUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method=%s, want GET", r.Method)
				}
				if r.URL.Path != "/v1/sms/status/MSG-123" {
					t.Fatalf("path=%s", r.URL.Path)
				}
				if got := r.Header.Get("Authorization"); got != "Basic token" {
					t.Fatalf("authorization=%q", got)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"success":true,"data":{"messageId":"MSG-123","status":"` + tt.native + `"}}`))
			}))
			defer server.Close()

			p, err := sendexa.New(sendexa.Config{Token: "token", BaseURL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			result, err := p.CheckSMSStatus(context.Background(), provider.SMSStatusRequest{
				Reference:         "attempt-123",
				ProviderMessageID: "MSG-123",
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.Reference != "attempt-123" || result.ProviderMessageID != "MSG-123" {
				t.Fatalf("result=%+v", result)
			}
			if result.ProviderStatus != tt.native || result.Status != tt.normalized {
				t.Fatalf("result=%+v", result)
			}
		})
	}
}

func TestCheckSMSStatusRequiresProviderMessageID(t *testing.T) {
	p, err := sendexa.New(sendexa.Config{Token: "token"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := p.CheckSMSStatus(context.Background(), provider.SMSStatusRequest{Reference: "attempt-123"})
	if err != sendexa.ErrInvalidRequest {
		t.Fatalf("err=%v, want ErrInvalidRequest", err)
	}
	if result.Reference != "attempt-123" || result.Status != provider.SMSUnknown {
		t.Fatalf("result=%+v", result)
	}
}

func scenario(status int, body string) relaytest.Factory[sms.Message] {
	return func(t *testing.T) (relaytest.Provider[sms.Message], sms.Message) {
		t.Helper()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		}))
		t.Cleanup(server.Close)
		p, err := sendexa.New(sendexa.Config{Token: "token", BaseURL: server.URL})
		if err != nil {
			t.Fatal(err)
		}
		return p, sms.Message{To: "+233241234567", From: "Acme", Text: "hello"}
	}
}
