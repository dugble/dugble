package vonage_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dugble/dugble/server/internal/relay/providers/vonage"
	"github.com/dugble/dugble/server/internal/relay/relaytest"
	"github.com/dugble/dugble/server/internal/relay/sms"
)

func TestProviderContract(t *testing.T) {
	relaytest.Contract[sms.Message]{
		Name:     "vonage",
		Accepted: vonageScenario(http.StatusAccepted, `{"message_uuid":"aaaaaaaa-bbbb-4ccc-8ddd-0123456789ab"}`),
		Rejected: vonageScenario(http.StatusUnprocessableEntity, `{"title":"Invalid Request","detail":"invalid to"}`),
		Unknown:  vonageScenario(http.StatusRequestTimeout, `{"title":"Request Timeout","detail":"request timed out"}`),
		RunRelay: func(ctx context.Context, primary, fallback relaytest.Provider[sms.Message], message sms.Message) (sms.SendResult, error) {
			router, err := sms.NewRelay(primary, fallback)
			if err != nil {
				return sms.SendResult{}, err
			}
			return router.Send(ctx, message)
		},
	}.Run(t)
}

func vonageScenario(status int, body string) relaytest.Factory[sms.Message] {
	return func(t *testing.T) (relaytest.Provider[sms.Message], sms.Message) {
		t.Helper()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		}))
		t.Cleanup(server.Close)
		provider, err := vonage.New(vonage.Config{APIKey: "api-key", APISecret: "secret", BaseURL: server.URL})
		if err != nil {
			t.Fatal(err)
		}
		return provider, sms.Message{To: "+233200000000", From: "Acme", Text: "hello"}
	}
}
