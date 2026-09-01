package lifecycle

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalidPlan          = errors.New("subscription plan is required")
	ErrInvalidPeriod        = errors.New("subscription period is invalid")
	ErrInvalidStatus        = errors.New("subscription status is invalid")
	ErrInvalidPendingChange = errors.New("pending plan change is invalid")
)

// Decide returns the transition to execute at the current period boundary.
// Payment outcomes are applied later by renewal after any required charge.
func Decide(state State, now time.Time) (Decision, error) {
	if state.PlanCode == "" {
		return Decision{}, ErrInvalidPlan
	}
	if state.CurrentPeriodStart.IsZero() || !state.CurrentPeriodEnd.After(state.CurrentPeriodStart) {
		return Decision{}, ErrInvalidPeriod
	}
	if err := validatePendingChange(state); err != nil {
		return Decision{}, err
	}
	switch state.Status {
	case StatusCanceled:
		return noTransition(state), nil
	case StatusActive, StatusPastDue:
	default:
		return Decision{}, fmt.Errorf("%w: %q", ErrInvalidStatus, state.Status)
	}
	if now.Before(state.CurrentPeriodEnd) {
		return noTransition(state), nil
	}
	if state.CancelAtPeriodEnd {
		return Decision{Transition: TransitionCancel, CurrentPlan: state.PlanCode, NextPlan: state.PlanCode, PeriodStart: state.CurrentPeriodStart, PeriodEnd: state.CurrentPeriodEnd}, nil
	}
	decision := Decision{Transition: TransitionRenew, CurrentPlan: state.PlanCode, NextPlan: state.PlanCode, PeriodStart: state.CurrentPeriodEnd, PeriodEnd: state.CurrentPeriodEnd.AddDate(0, 1, 0), ChargeRequired: true}
	if state.PendingPlanCode != nil {
		decision.Transition = TransitionChangePlan
		decision.NextPlan = *state.PendingPlanCode
	}
	return decision, nil
}
func validatePendingChange(state State) error {
	if state.PendingPlanCode == nil && state.PendingPlanEffectiveAt == nil {
		return nil
	}
	if state.PendingPlanCode == nil || state.PendingPlanEffectiveAt == nil || *state.PendingPlanCode == "" || *state.PendingPlanCode == state.PlanCode || !state.PendingPlanEffectiveAt.Equal(state.CurrentPeriodEnd) {
		return ErrInvalidPendingChange
	}
	return nil
}
func noTransition(state State) Decision {
	return Decision{Transition: TransitionNone, CurrentPlan: state.PlanCode, NextPlan: state.PlanCode, PeriodStart: state.CurrentPeriodStart, PeriodEnd: state.CurrentPeriodEnd}
}
