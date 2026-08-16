package mnotify_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dugble/dugble/server/internal/providers/mnotify"
	"github.com/dugble/dugble/server/internal/relay/relaytest"
	"github.com/dugble/dugble/server/internal/relay/sms"
)

func TestProviderContract(t *testing.T) {
	relaytest.Contract[sms.Message]{
		Name:     "mnotify",
		Accepted: scenario(http.StatusOK, `{"status":"success","code":2000,"summary":{"message_id":"msg-1"}}`),
		Unknown:  scenario(http.StatusBadRequest, `{"status":"error","code":4000,"message":"invalid request"}`),
		RunRelay: func(ctx context.Context, primary, fallback relaytest.Provider[sms.Message], message sms.Message) (sms.SendResult, error) {
			r, err := sms.NewRelay(primary, fallback)
			if err != nil {
				return sms.SendResult{}, err
			}
			return r.Send(ctx, message)
		},
	}.Run(t)
}

func scenario(status int, body string) relaytest.Factory[sms.Message] {
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
