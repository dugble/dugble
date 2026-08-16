package moolre_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	provider "github.com/dugble/dugble/server/internal/providers"
	"github.com/dugble/dugble/server/internal/providers/moolre"
)

func TestCreateSenderIDUsesHTTPContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s, want POST", r.Method)
		}
		if r.URL.Path != "/open/sms/query" {
			t.Fatalf("path=%s, want /open/sms/query", r.URL.Path)
		}
		if got := r.Header.Get("X-API-VASKEY"); got != "vas-secret" {
			t.Fatalf("X-API-VASKEY=%q, want vas-secret", got)
		}

		var body struct {
			Type      int `json:"type"`
			SenderIDs []struct {
				SenderID string `json:"senderid"`
				Approve  *bool  `json:"approve"`
			} `json:"senderids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Type != 3 {
			t.Fatalf("type=%d, want 3", body.Type)
		}
		if len(body.SenderIDs) != 1 {
			t.Fatalf("senderids=%+v, want one sender ID", body.SenderIDs)
		}
		if body.SenderIDs[0].SenderID != "ACME" {
			t.Fatalf("senderid=%q, want ACME", body.SenderIDs[0].SenderID)
		}
		if body.SenderIDs[0].Approve != nil {
			t.Fatalf("approve=%v, want omitted", *body.SenderIDs[0].Approve)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":1,"code":"ASMQ12","message":"Sender IDs Created Successfully.","data":null,"go":null}`))
	}))
	defer server.Close()

	p, err := moolre.New(moolre.Config{VASKey: "vas-secret", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}

	result, err := p.CreateSenderID(context.Background(), provider.CreateSenderIDRequest{SenderID: "ACME"})
	if err != nil {
		t.Fatal(err)
	}
	if result.SenderID != "ACME" {
		t.Fatalf("sender ID=%q, want ACME", result.SenderID)
	}
	if result.Status != provider.SenderIDPending {
		t.Fatalf("status=%s, want pending", result.Status)
	}
	if result.ProviderCode != "ASMQ12" {
		t.Fatalf("provider code=%q, want ASMQ12", result.ProviderCode)
	}
}

func TestCreateSenderIDPermissionDeniedPreservesProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status":0,"code":"ASMQ09","message":"You do not have permission to approve Sender IDs. Contact Customer Support.","data":"senderid","go":null}`))
	}))
	defer server.Close()

	p, err := moolre.New(moolre.Config{VASKey: "vas-secret", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}

	result, err := p.CreateSenderID(context.Background(), provider.CreateSenderIDRequest{SenderID: "ACME"})
	if err == nil {
		t.Fatal("expected permission error")
	}
	if result.Status != provider.SenderIDUnknown {
		t.Fatalf("status=%s, want unknown", result.Status)
	}
	if result.ProviderCode != "ASMQ09" {
		t.Fatalf("provider code=%q, want ASMQ09", result.ProviderCode)
	}

	var apiErr *moolre.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error=%T, want *moolre.APIError", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("status code=%d, want 400", apiErr.StatusCode)
	}
	if apiErr.Code != "ASMQ09" {
		t.Fatalf("code=%q, want ASMQ09", apiErr.Code)
	}
	if apiErr.Message != "You do not have permission to approve Sender IDs. Contact Customer Support." {
		t.Fatalf("message=%q", apiErr.Message)
	}
}
