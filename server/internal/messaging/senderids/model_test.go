package senderid

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSenderIDJSONOmitsProvider(t *testing.T) {
	t.Parallel()

	provider := "moolre"
	payload, err := json.Marshal(SenderID{
		ID:          "sender-id",
		TeamID:      "team-id",
		Name:        "Leamout",
		CountryCode: "GH",
		Purpose:     "Transactional SMS",
		Status:      StatusApproved,
		Provider:    &provider,
	})
	if err != nil {
		t.Fatalf("marshal sender ID: %v", err)
	}

	if strings.Contains(string(payload), `"provider"`) || strings.Contains(string(payload), provider) {
		t.Fatalf("public sender ID JSON exposed provider: %s", payload)
	}
}
