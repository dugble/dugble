package subscription

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestChargeJSONIncludesCommunicationCredit(t *testing.T) {
	t.Parallel()
	payload, err := json.Marshal(Charge{
		ID: "charge-1",
		CommunicationCredit: &CommunicationCredit{
			ID:             "credit-1",
			GrantedUnits:   349_000_000,
			ConsumedUnits:  165_000_000,
			RemainingUnits: 184_000_000,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	body := string(payload)
	for _, expected := range []string{
		`"communication_credit"`,
		`"granted_units":349000000`,
		`"consumed_units":165000000`,
		`"remaining_units":184000000`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("JSON = %s, missing %s", body, expected)
		}
	}
}
