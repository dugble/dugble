package smscampaign

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestValidateCreateNormalizesRateControls(t *testing.T) {
	t.Parallel()
	_, _, request, err := validateCreate(CreateRequest{
		Name: "Campaign", SegmentID: validID, SenderID: validID, Body: "Hello",
	})
	if err != nil {
		t.Fatalf("validateCreate() error = %v", err)
	}
	if request.RateLimitPerSecond != 10 {
		t.Fatalf("rate limit = %d, want 10", request.RateLimitPerSecond)
	}
	invalid := request
	invalid.RateLimitPerSecond = 1001
	if _, _, _, err = validateCreate(invalid); err == nil {
		t.Fatal("validateCreate() accepted an excessive rate limit")
	}
}

func TestAnalyticsJSONContract(t *testing.T) {
	t.Parallel()
	payload, err := json.Marshal(Analytics{CampaignID: "campaign-1", Delivered: 4, ActualSegments: 6, ActualChargeUnits: 120})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	body := string(payload)
	for _, expected := range []string{`"delivered":4`, `"actual_segments":6`, `"actual_charge_units":120`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("JSON = %s, missing %s", body, expected)
		}
	}
}

func TestRecipientDeliveryJSONContract(t *testing.T) {
	t.Parallel()
	deliveryStatus := "delivered"
	deliveredAt := time.Date(2026, time.August, 9, 22, 2, 54, 983863000, time.UTC)
	recipient := Recipient{
		ID:             "419bc927-e86d-423a-9cb3-e76da2176ffa",
		Status:         "queued",
		DeliveryStatus: &deliveryStatus,
		DeliveredAt:    &deliveredAt,
	}

	encoded, err := json.Marshal(recipient)
	if err != nil {
		t.Fatalf("marshal recipient: %v", err)
	}
	body := string(encoded)
	for _, expected := range []string{
		`"status":"queued"`,
		`"delivery_status":"delivered"`,
		`"delivered_at":"2026-08-09T22:02:54.983863Z"`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("recipient JSON = %s, missing %s", body, expected)
		}
	}
}

func TestRecipientDeliveryFieldsAreOmittedWhenUnavailable(t *testing.T) {
	t.Parallel()
	encoded, err := json.Marshal(Recipient{Status: "excluded"})
	if err != nil {
		t.Fatalf("marshal recipient: %v", err)
	}
	body := string(encoded)
	for _, unexpected := range []string{`"delivery_status"`, `"delivered_at"`} {
		if strings.Contains(body, unexpected) {
			t.Fatalf("recipient JSON = %s, unexpectedly includes %s", body, unexpected)
		}
	}
}
