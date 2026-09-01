package hubtel

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/dugble/dugble/server/internal/platform/config"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestInitiateCheckoutUsesBasicAuthAndMerchantAccount(t *testing.T) {
	t.Parallel()

	client := NewClient(config.HubtelConfig{ClientID: "client-id", ClientSecret: "client-secret", MerchantAccountNumber: "11684"})
	client.HTTPClient = &http.Client{Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", request.Method)
		}
		if request.URL.Host != "payproxyapi.hubtel.com" || request.URL.Path != "/items/initiate" {
			t.Fatalf("url = %s", request.URL.String())
		}
		wantAuthorization := "Basic " + base64.StdEncoding.EncodeToString([]byte("client-id:client-secret"))
		if got := request.Header.Get("Authorization"); got != wantAuthorization {
			t.Fatalf("Authorization = %q, want %q", got, wantAuthorization)
		}
		var payload InitiateCheckoutRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.MerchantAccountNumber != "11684" || payload.TotalAmount != 100 {
			t.Fatalf("payload = %#v", payload)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"responseCode":"0000","status":"Success","data":{"checkoutUrl":"https://pay.hubtel.com/checkout-id","checkoutId":"checkout-id","clientReference":"inv0012","message":"","checkoutDirectUrl":"https://pay.hubtel.com/checkout-id/direct"}}`)),
			Header:     make(http.Header),
		}, nil
	})}

	result, err := client.InitiateCheckout(context.Background(), InitiateCheckoutRequest{
		TotalAmount:     100,
		Description:     "Dugble wallet top-up",
		CallbackURL:     "https://api.dugble.com/payments/hubtel/callback",
		ReturnURL:       "https://dugble.com/dashboard/billing/transactions",
		CancellationURL: "https://dugble.com/dashboard/billing/transactions",
		ClientReference: "inv0012",
	})
	if err != nil {
		t.Fatalf("InitiateCheckout() error = %v", err)
	}
	if result.Data.CheckoutID != "checkout-id" || result.Data.ClientReference != "inv0012" {
		t.Fatalf("result = %#v", result)
	}
}

func TestCheckTransactionStatusPreservesAmounts(t *testing.T) {
	t.Parallel()

	client := NewClient(config.HubtelConfig{ClientID: "client-id", ClientSecret: "client-secret", MerchantAccountNumber: "11684"})
	client.HTTPClient = &http.Client{Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", request.Method)
		}
		if request.URL.Host != "api-txnstatus.hubtel.com" || request.URL.Path != "/transactions/11684/status" {
			t.Fatalf("url = %s", request.URL.String())
		}
		if got := request.URL.Query().Get("clientReference"); got != "inv0012" {
			t.Fatalf("clientReference = %q, want inv0012", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"message":"Successful","responseCode":"0000","data":{"date":"2024-04-25T21:45:48.4740964Z","status":"Paid","transactionId":"txn-id","externalTransactionId":"external-id","paymentMethod":"mobilemoney","clientReference":"inv0012","currencyCode":null,"amount":0.1,"charges":0.02,"amountAfterCharges":0.08,"isFulfilled":null}}`)),
			Header:     make(http.Header),
		}, nil
	})}

	result, err := client.CheckTransactionStatus(context.Background(), "inv0012")
	if err != nil {
		t.Fatalf("CheckTransactionStatus() error = %v", err)
	}
	if result.Data.Status != "Paid" || result.Data.Amount != 0.1 || result.Data.Charges != 0.02 || result.Data.AmountAfterCharges != 0.08 {
		t.Fatalf("result = %#v", result)
	}
}
