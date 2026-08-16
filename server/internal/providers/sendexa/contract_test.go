package sendexa_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dugble/dugble/server/internal/providers/sendexa"
	"github.com/dugble/dugble/server/internal/relay/relaytest"
	"github.com/dugble/dugble/server/internal/relay/sms"
)

func TestProviderContract(t *testing.T) {
	relaytest.Contract[sms.Message]{
		Name:     "sendexa",
		Accepted: scenario(http.StatusOK, `{"data":{"messageId":"MSG-1","delivery":{"status":"PENDING"}}}`),
		Unknown:  scenario(http.StatusBadRequest, `{"data":{"delivery":{"status":"FAILED"}}}`),
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
		_, _ = w.Write([]byte(`{"data":{"messageId":"MSG-cd57528e-60f3-4508-a6ef-e73982c6c78e","delivery":{"status":"SENT"}}}`))
	}))
	defer server.Close()

	provider, err := sendexa.New(sendexa.Config{Token: "token", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}

	result, err := provider.Send(context.Background(), sms.Message{
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

func scenario(status int, body string) relaytest.Factory[sms.Message] {
	return func(t *testing.T) (relaytest.Provider[sms.Message], sms.Message) {
		t.Helper()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		}))
		t.Cleanup(server.Close)
		provider, err := sendexa.New(sendexa.Config{Token: "token", BaseURL: server.URL})
		if err != nil {
			t.Fatal(err)
		}
		return provider, sms.Message{To: "+233241234567", From: "Acme", Text: "hello"}
	}
}
