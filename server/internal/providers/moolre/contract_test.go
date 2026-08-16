package moolre_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dugble/dugble/server/internal/providers/moolre"
	"github.com/dugble/dugble/server/internal/relay/relaytest"
	"github.com/dugble/dugble/server/internal/relay/sms"
)

func TestProviderContract(t *testing.T) {
	relaytest.Contract[sms.Message]{
		Name:     "moolre",
		Accepted: scenario(http.StatusOK, `{"status":1,"code":"SMS01","message":"Success"}`),
		Rejected: scenario(http.StatusBadRequest, `{"status":0,"code":"ASMS07","message":"Sender ID is not approved"}`),
		Unknown:  scenario(http.StatusInternalServerError, `{"status":0,"code":"ERR","message":"try later"}`),
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
		provider, err := moolre.NewTestProvider(moolre.Config{VASKey: "vas-secret"}, server.URL)
		if err != nil {
			t.Fatal(err)
		}
		return provider, sms.Message{To: "0241234567", From: "Acme", Text: "hello"}
	}
}
