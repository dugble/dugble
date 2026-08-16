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

func TestCheckSenderIDStatusUsesHTTPContract(t *testing.T) {
	cases := []struct {
		name     string
		senderID string
		approval string
		want     provider.SenderIDStatus
	}{
		{name: "approved", senderID: "SmartSMS", approval: "Approved", want: provider.SenderIDActive},
		{name: "pending", senderID: "Dummy", approval: "Pending", want: provider.SenderIDPending},
		{name: "rejected", senderID: "Momo", approval: "Rejected", want: provider.SenderIDRejected},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
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
					Type     int    `json:"type"`
					SenderID string `json:"senderid"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body.Type != 1 {
					t.Fatalf("type=%d, want 1", body.Type)
				}
				if body.SenderID != tc.senderID {
					t.Fatalf("senderid=%q, want %q", body.SenderID, tc.senderID)
				}

				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(map[string]any{
					"status":  1,
					"code":    "ASMQ01",
					"message": "Sender ID Status",
					"data": map[string]any{
						"senderid":    tc.senderID,
						"approval":    tc.approval,
						"whitelisted": false,
					},
					"go": nil,
				}); err != nil {
					t.Fatal(err)
				}
			}))
			defer server.Close()

			p, err := moolre.New(moolre.Config{VASKey: "vas-secret", BaseURL: server.URL})
			if err != nil {
				t.Fatal(err)
			}

			result, err := p.CheckSenderIDStatus(context.Background(), provider.SenderIDStatusRequest{SenderID: tc.senderID})
			if err != nil {
				t.Fatal(err)
			}
			if result.SenderID != tc.senderID {
				t.Fatalf("sender ID=%q, want %q", result.SenderID, tc.senderID)
			}
			if result.Status != tc.want {
				t.Fatalf("status=%s, want %s", result.Status, tc.want)
			}
			if result.ProviderStatus != tc.approval {
				t.Fatalf("provider status=%q, want %q", result.ProviderStatus, tc.approval)
			}
			if result.ProviderCode != "ASMQ01" {
				t.Fatalf("provider code=%q, want ASMQ01", result.ProviderCode)
			}
		})
	}
}
