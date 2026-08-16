package mnotify_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dugble/relay/providers/mnotify"
	"github.com/dugble/relay/relaytest"
	"github.com/dugble/relay/sms"
)

func TestProviderContract(t *testing.T) {
	relaytest.Contract[sms.Message]{
		Name:     "mnotify",
		Accepted: mnotifyScenario(http.StatusOK, `{"status":"success","code":2000,"summary":{"message_id":"msg-1"}}`),
		Unknown:  mnotifyScenario(http.StatusBadRequest, `{"status":"error","code":4000,"message":"invalid request"}`),
		RunRelay: func(ctx context.Context, primary, fallback relaytest.Provider[sms.Message], message sms.Message) (sms.SendResult, error) {
			router, err := sms.NewRelay(primary, fallback)
			if err != nil {
				return sms.SendResult{}, err
			}
			return router.Send(ctx, message)
		},
	}.Run(t)
}

func mnotifyScenario(status int, body string) relaytest.Factory[sms.Message] {
	return func(t *testing.T) (relaytest.Provider[sms.Message], sms.Message) {
		t.Helper()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		}))
		t.Cleanup(server.Close)
		provider, err := mnotify.New(mnotify.Config{APIKey: "secret", BaseURL: server.URL})
		if err != nil {
			t.Fatal(err)
		}
		return provider, sms.Message{To: "0241234567", From: "Acme", Text: "hello"}
	}
}
