package hubtel_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dugble/relay/providers/hubtel"
	"github.com/dugble/relay/relaytest"
	"github.com/dugble/relay/sms"
)

func TestProviderContract(t *testing.T) {
	relaytest.Contract[sms.Message]{
		Name:     "hubtel",
		Accepted: hubtelScenario(http.StatusCreated, `{"data":{"messageId":"msg-1"}}`),
		Rejected: hubtelScenario(http.StatusTooManyRequests, `{"message":"rate limited"}`),
		Unknown:  hubtelScenario(http.StatusInternalServerError, `{"message":"internal error"}`),
		RunRelay: func(ctx context.Context, primary, fallback relaytest.Provider[sms.Message], message sms.Message) (sms.SendResult, error) {
			router, err := sms.NewRelay(primary, fallback)
			if err != nil {
				return sms.SendResult{}, err
			}
			return router.Send(ctx, message)
		},
	}.Run(t)
}

func hubtelScenario(status int, body string) relaytest.Factory[sms.Message] {
	return func(t *testing.T) (relaytest.Provider[sms.Message], sms.Message) {
		t.Helper()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		}))
		t.Cleanup(server.Close)
		provider, err := hubtel.New(hubtel.Config{ClientID: "client", ClientSecret: "secret", BaseURL: server.URL})
		if err != nil {
			t.Fatal(err)
		}
		return provider, sms.Message{To: "+233200000000", From: "Acme", Text: "hello"}
	}
}
