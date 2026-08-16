package moolre_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	provider "github.com/dugble/dugble/server/internal/providers"
	"github.com/dugble/dugble/server/internal/providers/moolre"
	"github.com/dugble/dugble/server/internal/relay/sms"
)

func TestSendIncludesReference(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/open/sms/send" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var body struct {
			Messages []struct {
				Ref string `json:"ref"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.Messages) != 1 || body.Messages[0].Ref != "attempt-123" {
			t.Fatalf("messages = %+v", body.Messages)
		}
		_, _ = w.Write([]byte(`{"status":1,"code":"SMS01","message":"Success"}`))
	}))
	defer server.Close()

	p, err := moolre.New(moolre.Config{VASKey: "secret", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := p.Send(context.Background(), sms.Message{Reference: "attempt-123", To: "0241234567", From: "Acme", Text: "hello"})
	if err != nil || result.State != sms.SubmissionAccepted {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestCreateSenderID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/open/sms/query" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("X-API-VASKEY"); got != "secret" {
			t.Fatalf("X-API-VASKEY = %q", got)
		}
		var body struct {
			Type      int `json:"type"`
			SenderIDs []struct {
				SenderID string `json:"senderid"`
			} `json:"senderids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Type != 3 || len(body.SenderIDs) != 1 || body.SenderIDs[0].SenderID != "ACME" {
			t.Fatalf("body = %+v", body)
		}
		_, _ = w.Write([]byte(`{"status":1,"code":"ASMQ12","message":"Sender IDs Created Successfully."}`))
	}))
	defer server.Close()

	p, err := moolre.New(moolre.Config{VASKey: "secret", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := p.CreateSenderID(context.Background(), provider.CreateSenderIDRequest{SenderID: "ACME"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != provider.SenderIDPending || result.ProviderCode != "ASMQ12" {
		t.Fatalf("result = %+v", result)
	}
}

func TestCheckSenderIDStatus(t *testing.T) {
	cases := []struct {
		approval string
		want     provider.SenderIDStatus
	}{
		{approval: "Approved", want: provider.SenderIDActive},
		{approval: "Pending", want: provider.SenderIDPending},
		{approval: "Rejected", want: provider.SenderIDRejected},
	}
	for _, tc := range cases {
		t.Run(tc.approval, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/open/sms/status" {
					t.Fatalf("path = %q", r.URL.Path)
				}
				var body struct {
					Type     int    `json:"type"`
					SenderID string `json:"senderid"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body.Type != 1 || body.SenderID != "ACME" {
					t.Fatalf("body = %+v", body)
				}
				_, _ = w.Write([]byte(`{"status":1,"code":"ASMQ01","message":"Sender ID Status","data":{"senderid":"ACME","approval":"` + tc.approval + `","whitelisted":false}}`))
			}))
			defer server.Close()

			p, err := moolre.New(moolre.Config{VASKey: "secret", BaseURL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			result, err := p.CheckSenderIDStatus(context.Background(), provider.SenderIDStatusRequest{SenderID: "ACME"})
			if err != nil {
				t.Fatal(err)
			}
			if result.Status != tc.want || result.ProviderStatus != tc.approval {
				t.Fatalf("result = %+v", result)
			}
		})
	}
}

func TestCheckSMSStatusPreservesNativeStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/open/sms/status" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var body struct {
			Type int      `json:"type"`
			Ref  []string `json:"ref"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Type != 5 || len(body.Ref) != 1 || body.Ref[0] != "attempt-123" {
			t.Fatalf("body = %+v", body)
		}
		_, _ = w.Write([]byte(`{"status":1,"code":"ASMQ10","message":"SMS Status","data":[{"ref":"attempt-123","status":3}]}`))
	}))
	defer server.Close()

	p, err := moolre.New(moolre.Config{VASKey: "secret", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := p.CheckSMSStatus(context.Background(), provider.SMSStatusRequest{Reference: "attempt-123"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != provider.SMSUnknown || result.ProviderStatus != "3" || result.ProviderCode != "ASMQ10" {
		t.Fatalf("result = %+v", result)
	}
}
