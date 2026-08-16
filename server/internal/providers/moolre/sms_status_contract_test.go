package moolre_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	provider "github.com/dugble/dugble/server/internal/providers"
	"github.com/dugble/dugble/server/internal/providers/moolre"
)

func TestCheckSMSStatusUsesHTTPContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s, want POST", r.Method)
		}
		if r.URL.Path != "/open/sms/status" {
			t.Fatalf("path=%s, want /open/sms/status", r.URL.Path)
		}
		if got := r.Header.Get("X-API-VASKEY"); got != "vas-secret" {
			t.Fatalf("X-API-VASKEY=%q, want vas-secret", got)
		}

		var body struct {
			Type int      `json:"type"`
			Ref  []string `json:"ref"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Type != 5 {
			t.Fatalf("type=%d, want 5", body.Type)
		}
		if len(body.Ref) != 1 || body.Ref[0] != "uuid--001" {
			t.Fatalf("ref=%v, want [uuid--001]", body.Ref)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":1,"code":"ASMQ10","message":"SMS Status","data":[{"ref":"0338954001737166274","status":3},{"ref":"uuid--001","status":2}],"go":null}`))
	}))
	defer server.Close()

	p, err := moolre.New(moolre.Config{VASKey: "vas-secret", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}

	result, err := p.CheckSMSStatus(context.Background(), provider.SMSStatusRequest{Reference: "uuid--001"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Reference != "uuid--001" {
		t.Fatalf("reference=%q, want uuid--001", result.Reference)
	}
	if result.Status != provider.SMSUnknown {
		t.Fatalf("status=%s, want unknown", result.Status)
	}
	if result.ProviderStatus != "2" {
		t.Fatalf("provider status=%q, want 2", result.ProviderStatus)
	}
	if result.ProviderCode != "ASMQ10" {
		t.Fatalf("provider code=%q, want ASMQ10", result.ProviderCode)
	}
}
