package twilio_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dugble/dugble/server/internal/relay/providers/twilio"
	"github.com/dugble/dugble/server/internal/relay/relaytest"
	"github.com/dugble/dugble/server/internal/relay/sms"
)

func TestProviderContract(t *testing.T) {
	relaytest.Contract[sms.Message]{
		Name:     "twilio",
		Accepted: twilioScenario(http.StatusCreated, `{"sid":"SM123","status":"queued"}`),
		Rejected: twilioScenario(http.StatusBadRequest, `{"code":21211,"message":"invalid to","status":400}`),
		Unknown:  twilioScenario(http.StatusRequestTimeout, `{"message":"request timed out","status":408}`),
		RunRelay: func(ctx context.Context, primary, fallback relaytest.Provider[sms.Message], message sms.Message) (sms.SendResult, error) {
			router, err := sms.NewRelay(primary, fallback)
			if err != nil {
				return sms.SendResult{}, err
			}
			return router.Send(ctx, message)
		},
	}.Run(t)
}

func twilioScenario(status int, body string) relaytest.Factory[sms.Message] {
	return func(t *testing.T) (relaytest.Provider[sms.Message], sms.Message) {
		t.Helper()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		}))
		t.Cleanup(server.Close)
		provider, err := twilio.New(twilio.Config{AccountSID: "AC123", AuthToken: "token", BaseURL: server.URL})
		if err != nil {
			t.Fatal(err)
		}
		return provider, sms.Message{To: "+233200000000", From: "Acme", Text: "hello"}
	}
}
