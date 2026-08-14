package renewal

import (
	"time"

	charges "github.com/dugble/dugble/server/internal/billing/charge/subscription"
	"github.com/dugble/dugble/server/internal/billing/subscription/lifecycle"
	"github.com/google/uuid"
)

type Due struct {
	SubscriptionID uuid.UUID
	TeamID         uuid.UUID
	State          lifecycle.State
}

type BillingRecipient struct {
	Name     string
	Email    string
	TeamName string
}

type Outcome string

const (
	OutcomeNotDue           Outcome = "not_due"
	OutcomeCanceled         Outcome = "canceled"
	OutcomeRenewed          Outcome = "renewed"
	OutcomePlanChanged      Outcome = "plan_changed"
	OutcomePastDue          Outcome = "past_due"
	OutcomePriceUnavailable Outcome = "price_unavailable"
)

type Result struct {
	SubscriptionID uuid.UUID      `json:"subscription_id"`
	TeamID         uuid.UUID      `json:"team_id"`
	Outcome        Outcome        `json:"outcome"`
	PreviousPlan   string         `json:"previous_plan"`
	CurrentPlan    string         `json:"current_plan"`
	PeriodStart    time.Time      `json:"period_start"`
	PeriodEnd      time.Time      `json:"period_end"`
	Charge         charges.Result `json:"charge"`
}
type BatchResult struct {
	Processed        int
	Renewed          int
	PlanChanged      int
	Canceled         int
	PastDue          int
	PriceUnavailable int
	Failures         []Failure
}

type Failure struct {
	TeamID uuid.UUID
	Err    error
}

func (result *BatchResult) Add(outcome Outcome) {
	result.Processed++
	switch outcome {
	case OutcomeRenewed:
		result.Renewed++
	case OutcomePlanChanged:
		result.PlanChanged++
	case OutcomeCanceled:
		result.Canceled++
	case OutcomePastDue:
		result.PastDue++
	case OutcomePriceUnavailable:
		result.PriceUnavailable++
	}
}

func (result *BatchResult) AddFailure(failure Failure) {
	result.Processed++
	result.Failures = append(result.Failures, failure)
}
