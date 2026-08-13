package subscriptions

import "time"

type Subscription struct {
	ID                     string     `json:"id"`
	TeamID                 string     `json:"team_id"`
	TeamName               string     `json:"team_name"`
	PlanCode               string     `json:"plan_code"`
	Status                 string     `json:"status"`
	BillingMarket          string     `json:"billing_market"`
	Currency               string     `json:"currency"`
	CurrentPeriodStart     time.Time  `json:"current_period_start"`
	CurrentPeriodEnd       time.Time  `json:"current_period_end"`
	PendingPlanCode        *string    `json:"pending_plan_code,omitempty"`
	PendingPlanEffectiveAt *time.Time `json:"pending_plan_effective_at,omitempty"`
	CancelAtPeriodEnd      bool       `json:"cancel_at_period_end"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}
type Charge struct {
	ID             string    `json:"id"`
	SubscriptionID string    `json:"subscription_id"`
	TeamID         string    `json:"team_id"`
	PlanCode       string    `json:"plan_code"`
	BillingMarket  string    `json:"billing_market"`
	Currency       string    `json:"currency"`
	PeriodStart    time.Time `json:"period_start"`
	PeriodEnd      time.Time `json:"period_end"`
	AmountUnits    int64     `json:"amount_units"`
	Status         string    `json:"status"`
	FailureCode    *string   `json:"failure_code,omitempty"`
	AttemptCount   int32     `json:"attempt_count"`
	ReferenceID    string    `json:"reference_id"`
	CreatedAt      time.Time `json:"created_at"`
}
type Filter struct {
	TeamID, Status string
	Limit, Offset  int32
}
type Page struct {
	Data    []Subscription `json:"data"`
	Limit   int32          `json:"limit"`
	Offset  int32          `json:"offset"`
	HasMore bool           `json:"has_more"`
}
type ChargePage struct {
	Data    []Charge `json:"data"`
	Limit   int32    `json:"limit"`
	Offset  int32    `json:"offset"`
	HasMore bool     `json:"has_more"`
}

type ActionInput struct {
	Reason      string `json:"reason"`
	ActorUserID string `json:"-"`
	SessionID   string `json:"-"`
}

type ChangePlanInput struct {
	PlanCode    string `json:"plan_code"`
	Reason      string `json:"reason"`
	ActorUserID string `json:"-"`
	SessionID   string `json:"-"`
}
