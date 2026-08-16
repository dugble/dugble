package postmark_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dugble/relay/email"
	"github.com/dugble/relay/providers/postmark"
	"github.com/dugble/relay/relaytest"
)

func TestProviderContract(t *testing.T) {
	relaytest.Contract[email.Message]{
		Name:     "postmark",
		Accepted: postmarkScenario(http.StatusOK, `{"ErrorCode":0,"Message":"OK","MessageID":"message-123"}`),
		Rejected: postmarkScenario(http.StatusUnprocessableEntity, `{"ErrorCode":300,"Message":"Invalid email request"}`),
		Unknown:  postmarkScenario(http.StatusServiceUnavailable, `{"ErrorCode":0,"Message":"Service unavailable"}`),
		RunRelay: func(ctx context.Context, primary, fallback relaytest.Provider[email.Message], message email.Message) (email.SendResult, error) {
			router, err := email.NewRelay(primary, fallback)
			if err != nil {
				return email.SendResult{}, err
			}
			return router.Send(ctx, message)
		},
	}.Run(t)
}

func postmarkScenario(status int, body string) relaytest.Factory[email.Message] {
	return func(t *testing.T) (relaytest.Provider[email.Message], email.Message) {
		t.Helper()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		}))
		t.Cleanup(server.Close)
		provider, err := postmark.New(postmark.Config{ServerToken: "server-token", BaseURL: server.URL})
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
