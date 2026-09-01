package smscampaign

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCostEstimateJSONContract(t *testing.T) {
	t.Parallel()
	currency := "GHS"
	payload, err := json.Marshal(CostEstimate{
		CampaignID: "campaign-1", Currency: &currency, Recipients: 2,
		EstimatedSegments: 3, EstimatedCostUnits: 240,
		EstimatedBillableCostUnits: 160, PreflightAllowanceSegments: 1,
		ActualSegments: 3, ActualChargeUnits: 160,
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	body := string(payload)
	for _, expected := range []string{
		`"currency":"GHS"`, `"estimated_segments":3`,
		`"estimated_cost_units":240`, `"estimated_billable_cost_units":160`,
		`"preflight_allowance_segments":1`, `"actual_charge_units":160`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("JSON = %s, missing %s", body, expected)
		}
	}
}
