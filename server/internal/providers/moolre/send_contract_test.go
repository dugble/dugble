package moolre_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dugble/dugble/server/internal/providers/moolre"
	"github.com/dugble/dugble/server/internal/relay/sms"
)

func TestSendUsesMoolreHTTPContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/open/sms/send" {
			t.Fatalf("path = %q, want /open/sms/send", r.URL.Path)
		}
		if got := r.Header.Get("X-API-VASKEY"); got != "vas-secret" {
			t.Fatalf("X-API-VASKEY = %q, want vas-secret", got)
		}

		var body struct {
			Type     int    `json:"type"`
			SenderID string `json:"senderid"`
			Messages []struct {
				Recipient string `json:"recipient"`
				Message   string `json:"message"`
				Ref       string `json:"ref"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Type != 1 {
			t.Fatalf("type = %d, want 1", body.Type)
		}
		if body.SenderID != "Acme" {
			t.Fatalf("senderid = %q, want Acme", body.SenderID)
		}
		if len(body.Messages) != 1 {
			t.Fatalf("messages length = %d, want 1", len(body.Messages))
		}
		message := body.Messages[0]
		if message.Recipient != "0241234567" || message.Message != "hello" || message.Ref != "attempt-123" {
			t.Fatalf("message = %+v", message)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":1,"code":"SMS01","message":"Success","data":null,"go":null}`))
	}))
	defer server.Close()

	provider, err := moolre.NewTestProvider(moolre.Config{VASKey: "vas-secret"}, server.URL)
	if err != nil {
		t.Fatal(err)
	}

	result, err := provider.Send(context.Background(), sms.Message{
		Reference: "attempt-123",
		To:        "0241234567",
		From:      "Acme",
		Text:      "hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.State != sms.SubmissionAccepted {
		t.Fatalf("state = %s, want accepted", result.State)
	}
}

func TestSendClassifiesDocumentedRejections(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
		body       string
		wantCode   string
		wantMsg    string
	}{
		{
			name:       "unapproved sender ID",
			statusCode: http.StatusBadRequest,
			body:       `{"status":0,"code":"ASMS07","message":"Sender ID is not approved, Please login on app.moolre.com and setup your Sender ID.","data":"senderid","go":null}`,
			wantCode:   "ASMS07",
			wantMsg:    "Sender ID is not approved, Please login on app.moolre.com and setup your Sender ID.",
		},
		{
			name:       "authentication error",
			statusCode: http.StatusUnauthorized,
			body:       `{"status":0,"code":"AIN01","message":"Authentication Error","data":null,"go":null}`,
			wantCode:   "AIN01",
			wantMsg:    "Authentication Error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			provider, err := moolre.NewTestProvider(moolre.Config{VASKey: "vas-secret"}, server.URL)
			if err != nil {
				t.Fatal(err)
			}
			result, err := provider.Send(context.Background(), sms.Message{To: "0241234567", From: "Acme", Text: "hello"})
			if result.State != sms.SubmissionRejected {
				t.Fatalf("state = %s, want rejected", result.State)
			}

			var apiErr *moolre.APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("error = %T %v, want *moolre.APIError", err, err)
			}
			if apiErr.StatusCode != tc.statusCode || apiErr.Code != tc.wantCode || apiErr.Message != tc.wantMsg {
				t.Fatalf("APIError = %+v", apiErr)
			}
		})
	}
}
