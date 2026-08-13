package lifecycle

import (
	"errors"
	"testing"
	"time"
)

var periodStart = time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
var periodEnd = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

func state(status Status) State {
	return State{Status: status, PlanCode: "growth", CurrentPeriodStart: periodStart, CurrentPeriodEnd: periodEnd}
}
func mustDecide(t *testing.T, s State, now time.Time) Decision {
	t.Helper()
	d, err := Decide(s, now)
	if err != nil {
		t.Fatal(err)
	}
	return d
}
func TestDecideNotDue(t *testing.T) {
	d := mustDecide(t, state(StatusActive), periodEnd.Add(-time.Second))
	if d.Transition != TransitionNone || d.ChargeRequired {
		t.Fatalf("%+v", d)
	}
}
func TestDecidePrioritizesCancellation(t *testing.T) {
	s := state(StatusActive)
	p := "scale"
	at := periodEnd
	s.PendingPlanCode = &p
	s.PendingPlanEffectiveAt = &at
	s.CancelAtPeriodEnd = true
	d := mustDecide(t, s, periodEnd)
	if d.Transition != TransitionCancel || d.NextPlan != "growth" || d.ChargeRequired || !d.PeriodStart.Equal(periodStart) {
		t.Fatalf("%+v", d)
	}
}
func TestDecideRenewsCurrentPlan(t *testing.T) {
	d := mustDecide(t, state(StatusActive), periodEnd)
	if d.Transition != TransitionRenew || d.NextPlan != "growth" || !d.ChargeRequired || !d.PeriodStart.Equal(periodEnd) || !d.PeriodEnd.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("%+v", d)
	}
}
func TestDecideActivatesPendingPlan(t *testing.T) {
	s := state(StatusActive)
	p := "scale"
	at := periodEnd
	s.PendingPlanCode = &p
	s.PendingPlanEffectiveAt = &at
	d := mustDecide(t, s, periodEnd)
	if d.Transition != TransitionChangePlan || d.CurrentPlan != "growth" || d.NextPlan != "scale" || !d.ChargeRequired {
		t.Fatalf("%+v", d)
	}
}
func TestDecideRetriesPastDuePeriod(t *testing.T) {
	d := mustDecide(t, state(StatusPastDue), periodEnd.Add(time.Hour))
	if d.Transition != TransitionRenew || !d.ChargeRequired || !d.PeriodStart.Equal(periodEnd) {
		t.Fatalf("%+v", d)
	}
}
func TestDecideCanceledIsNoOp(t *testing.T) {
	d := mustDecide(t, state(StatusCanceled), periodEnd.AddDate(0, 1, 0))
	if d.Transition != TransitionNone || d.ChargeRequired {
		t.Fatalf("%+v", d)
	}
}
func TestDecideRejectsInvalidPendingChange(t *testing.T) {
	s := state(StatusActive)
	p := "scale"
	at := periodEnd.AddDate(0, 1, 0)
	s.PendingPlanCode = &p
	s.PendingPlanEffectiveAt = &at
	_, err := Decide(s, periodEnd)
	if !errors.Is(err, ErrInvalidPendingChange) {
		t.Fatalf("%v", err)
	}
}
