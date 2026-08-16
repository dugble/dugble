package resend_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dugble/relay/email"
	"github.com/dugble/relay/providers/resend"
	"github.com/dugble/relay/relaytest"
)

func TestProviderContract(t *testing.T) {
	relaytest.Contract[email.Message]{
		Name:     "resend",
		Accepted: resendScenario(http.StatusOK, `{"id":"email-123"}`),
		Rejected: resendScenario(http.StatusUnprocessableEntity, `{"name":"invalid_from_address","message":"Invalid from"}`),
		Unknown:  resendScenario(http.StatusConflict, `{"name":"concurrent_idempotent_requests","message":"Request is in progress"}`),
		RunRelay: func(ctx context.Context, primary, fallback relaytest.Provider[email.Message], message email.Message) (email.SendResult, error) {
			router, err := email.NewRelay(primary, fallback)
			if err != nil {
				return email.SendResult{}, err
			}
			return router.Send(ctx, message)
		},
	}.Run(t)
}

func resendScenario(status int, body string) relaytest.Factory[email.Message] {
	return func(t *testing.T) (relaytest.Provider[email.Message], email.Message) {
		t.Helper()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		}))
		t.Cleanup(server.Close)
		provider, err := resend.New(resend.Config{APIKey: "re_test", BaseURL: server.URL})
		if err != nil {
			t.Fatal(err)
		}
		return provider, email.Message{
			From:    email.Address{Email: "sender@example.com", Name: "Acme"},
			To:      []email.Address{{Email: "customer@example.com"}},
			Subject: "Receipt",
			Text:    "thanks",
		}
	}
}
