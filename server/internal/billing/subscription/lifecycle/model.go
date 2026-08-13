package lifecycle

import "time"

type Status string

const (
	StatusActive   Status = "active"
	StatusPastDue  Status = "past_due"
	StatusCanceled Status = "canceled"
)

type State struct {
	Status                 Status
	PlanCode               string
	PendingPlanCode        *string
	PendingPlanEffectiveAt *time.Time
	CancelAtPeriodEnd      bool
	CurrentPeriodStart     time.Time
	CurrentPeriodEnd       time.Time
}
type Transition string

const (
	TransitionNone       Transition = "none"
	TransitionRenew      Transition = "renew"
	TransitionChangePlan Transition = "change_plan"
	TransitionCancel     Transition = "cancel"
)

type Decision struct {
	Transition     Transition
	CurrentPlan    string
	NextPlan       string
	PeriodStart    time.Time
	PeriodEnd      time.Time
	ChargeRequired bool
}
