package renewal

import (
	"context"
	"testing"
	"time"

	charges "github.com/coffeyvidzro/dugble/server/internal/billing/charge/subscription"
	"github.com/coffeyvidzro/dugble/server/internal/billing/subscription/lifecycle"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type storeStub struct {
	due     Due
	applied appliedState
}

func (s storeStub) GetDue(context.Context, pgx.Tx, uuid.UUID) (Due, error) { return s.due, nil }
func (s storeStub) ApplyCancellation(context.Context, pgx.Tx, uuid.UUID) (timeRange, error) {
	return timeRange{Start: s.due.State.CurrentPeriodStart, End: s.due.State.CurrentPeriodEnd}, nil
}
func (s storeStub) ApplyCharge(context.Context, pgx.Tx, uuid.UUID, bool, string, time.Time, time.Time) (appliedState, error) {
	return s.applied, nil
}

type chargerStub struct{ result charges.Result }

func (s chargerStub) ChargePeriod(context.Context, pgx.Tx, charges.Input) (charges.Result, error) {
	return s.result, nil
}
func TestServiceReturnsRenewedCharge(t *testing.T) {
	teamID, subscriptionID := uuid.New(), uuid.New()
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	next := end.AddDate(0, 1, 0)
	store := storeStub{due: Due{SubscriptionID: subscriptionID, TeamID: teamID, State: lifecycle.State{Status: lifecycle.StatusActive, PlanCode: "growth", CurrentPeriodStart: start, CurrentPeriodEnd: end}}, applied: appliedState{Plan: "growth", Period: timeRange{Start: end, End: next}}}
	service := NewService(store, chargerStub{charges.Result{Outcome: charges.OutcomeApplied, Status: "applied"}}, lifecycle.NewService())
	service.now = func() time.Time { return end }
	result, err := service.ProcessTeam(context.Background(), nil, teamID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != OutcomeRenewed || result.Charge.Status != "applied" || !result.PeriodStart.Equal(end) {
		t.Fatalf("%+v", result)
	}
}
