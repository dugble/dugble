package subscription

import (
	"time"

	"github.com/google/uuid"
)

type Outcome string

const (
	OutcomeApplied             Outcome = "applied"
	OutcomeAlreadyApplied      Outcome = "already_applied"
	OutcomeInsufficientBalance Outcome = "insufficient_balance"
	OutcomePriceUnavailable    Outcome = "price_unavailable"
)

type Input struct {
	SubscriptionID uuid.UUID
	TeamID         uuid.UUID
	PlanCode       string
	PeriodStart    time.Time
	PeriodEnd      time.Time
}

type Result struct {
	ChargeID         *uuid.UUID `json:"charge_id,omitempty"`
	Outcome          Outcome    `json:"outcome"`
	Status           string     `json:"status"`
	FailureCode      *string    `json:"failure_code,omitempty"`
	AttemptCount     int32      `json:"attempt_count"`
	LastAttemptedAt  *time.Time `json:"last_attempted_at,omitempty"`
	AppliedAt        *time.Time `json:"applied_at,omitempty"`
	PlanCode         string     `json:"plan_code"`
	Currency         string     `json:"currency"`
	AmountUnits      int64      `json:"amount_units"`
	RemainingBalance int64      `json:"remaining_balance"`
	CreditID         *uuid.UUID `json:"credit_id,omitempty"`
	CreditGranted    int64      `json:"credit_granted_units"`
}
