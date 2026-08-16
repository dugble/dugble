package teamtoken

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCreatedTokenJSONIncludesTokenFieldsAndSecret(t *testing.T) {
	createdAt := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	payload, err := json.Marshal(CreatedToken{
		Token: Token{
			ID:          "token-id",
			TeamID:      "team-id",
			Name:        "Deploy token",
			TokenPrefix: "dgb_team_prefix",
			Permissions: []string{"team:read"},
			CreatedAt:   createdAt,
			UpdatedAt:   createdAt,
		},
		Secret: "dgb_team_secret",
	})
	if err != nil {
		t.Fatalf("json.Marshal(CreatedToken) returned error: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("json.Unmarshal(CreatedToken) returned error: %v", err)
	}

	for _, key := range []string{"id", "team_id", "name", "token_prefix", "permissions", "created_at", "updated_at", "secret"} {
		if _, ok := body[key]; !ok {
			t.Fatalf("CreatedToken JSON missing %q: %s", key, payload)
		}
	}
}
