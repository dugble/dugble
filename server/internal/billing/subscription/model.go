package subscription

import "time"

type Subscription struct {
	ID                     string     `json:"id"`
	TeamID                 string     `json:"team_id"`
	PlanCode               string     `json:"plan_code"`
	Status                 string     `json:"status"`
	CurrentPeriodStart     time.Time  `json:"current_period_start"`
	CurrentPeriodEnd       time.Time  `json:"current_period_end"`
	PendingPlanCode        *string    `json:"pending_plan_code,omitempty"`
	PendingPlanEffectiveAt *time.Time `json:"pending_plan_effective_at,omitempty"`
	CancelAtPeriodEnd      bool       `json:"cancel_at_period_end"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

type SelectPlanInput struct {
	Plan string `json:"plan"`
}

type CommunicationCredit struct {
	ID             string `json:"id"`
	GrantedUnits   int64  `json:"granted_units"`
	ConsumedUnits  int64  `json:"consumed_units"`
	RemainingUnits int64  `json:"remaining_units"`
}

type Charge struct {
	ID                  string               `json:"id"`
	SubscriptionID      string               `json:"subscription_id"`
	PlanPriceID         string               `json:"plan_price_id"`
	PlanCode            string               `json:"plan_code"`
	BillingMarket       string               `json:"billing_market"`
	Currency            string               `json:"currency"`
	PeriodStart         time.Time            `json:"period_start"`
	PeriodEnd           time.Time            `json:"period_end"`
	AmountUnits         int64                `json:"amount_units"`
	Status              string               `json:"status"`
	FailureCode         *string              `json:"failure_code,omitempty"`
	AttemptCount        int32                `json:"attempt_count"`
	LastAttemptedAt     time.Time            `json:"last_attempted_at"`
	AppliedAt           *time.Time           `json:"applied_at,omitempty"`
	ReferenceID         string               `json:"reference_id"`
	CommunicationCredit *CommunicationCredit `json:"communication_credit,omitempty"`
	CreatedAt           time.Time            `json:"created_at"`
}

type ChargePage struct {
	Charges []Charge `json:"charges"`
	Limit   int32    `json:"limit"`
	Offset  int32    `json:"offset"`
}
