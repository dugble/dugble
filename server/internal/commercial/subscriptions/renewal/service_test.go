package renewal

import (
	"context"
	"testing"
	"time"

	charges "github.com/dugble/dugble/server/internal/commercial/charges/subscription"
	"github.com/dugble/dugble/server/internal/commercial/subscriptions/lifecycle"
	"github.com/dugble/dugble/server/internal/messaging/email/systemmail"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type storeStub struct {
	due        Due
	applied    appliedState
	recipients []BillingRecipient
}

func (s storeStub) ListBillingRecipients(context.Context, pgx.Tx, uuid.UUID) ([]BillingRecipient, error) {
	return s.recipients, nil
}

func (s storeStub) GetDue(context.Context, pgx.Tx, uuid.UUID) (Due, error) { return s.due, nil }
func (s storeStub) ApplyCancellation(context.Context, pgx.Tx, uuid.UUID) (timeRange, error) {
	return timeRange{Start: s.due.State.CurrentPeriodStart, End: s.due.State.CurrentPeriodEnd}, nil
}
func (s storeStub) ApplyCharge(context.Context, pgx.Tx, uuid.UUID, bool, string, time.Time, time.Time) (appliedState, error) {
	return s.applied, nil
}

type notifierStub struct {
	inputs []systemmail.SendSubscriptionPastDueInput
}

func (s *notifierStub) SendSubscriptionPastDue(_ context.Context, _ pgx.Tx, input systemmail.SendSubscriptionPastDueInput) error {
	s.inputs = append(s.inputs, input)
	return nil
}

func TestServiceNotifiesOwnerWhenSubscriptionFirstBecomesPastDue(t *testing.T) {
	teamID, subscriptionID := uuid.New(), uuid.New()
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	store := storeStub{
		due:        Due{SubscriptionID: subscriptionID, TeamID: teamID, State: lifecycle.State{Status: lifecycle.StatusActive, PlanCode: "growth", CurrentPeriodStart: start, CurrentPeriodEnd: end}},
		applied:    appliedState{Plan: "growth", Period: timeRange{Start: start, End: end}},
		recipients: []BillingRecipient{{Name: "Ada", Email: "ada@example.com", TeamName: "Acme"}},
	}
	notifier := &notifierStub{}
	charge := charges.Result{Outcome: charges.OutcomeInsufficientBalance, Status: "failed", PlanCode: "growth", Currency: "GHS", AmountUnits: 34900, RemainingBalance: 100}
	service := NewService(store, chargerStub{charge}, lifecycle.NewService()).WithPastDueNotifier(notifier)
	service.now = func() time.Time { return end }

	result, err := service.ProcessTeam(context.Background(), nil, teamID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != OutcomePastDue || len(notifier.inputs) != 1 {
		t.Fatalf("result=%+v notifications=%d", result, len(notifier.inputs))
	}
	input := notifier.inputs[0]
	if input.ToEmail != "ada@example.com" || input.TeamName != "Acme" || input.AmountUnits != 34900 || input.BalanceUnits != 100 {
		t.Fatalf("notification=%+v", input)
	}
}

func TestServiceDoesNotRepeatPastDueNotification(t *testing.T) {
	teamID := uuid.New()
	end := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	store := storeStub{
		due:        Due{SubscriptionID: uuid.New(), TeamID: teamID, State: lifecycle.State{Status: lifecycle.StatusPastDue, PlanCode: "growth", CurrentPeriodStart: end.AddDate(0, -1, 0), CurrentPeriodEnd: end}},
		applied:    appliedState{Plan: "growth", Period: timeRange{Start: end.AddDate(0, -1, 0), End: end}},
		recipients: []BillingRecipient{{Email: "owner@example.com"}},
	}
	notifier := &notifierStub{}
	service := NewService(store, chargerStub{charges.Result{Outcome: charges.OutcomeInsufficientBalance}}, lifecycle.NewService()).WithPastDueNotifier(notifier)
	service.now = func() time.Time { return end.Add(time.Hour) }
	if _, err := service.ProcessTeam(context.Background(), nil, teamID); err != nil {
		t.Fatal(err)
	}
	if len(notifier.inputs) != 0 {
		t.Fatalf("notifications=%d, want 0", len(notifier.inputs))
	}
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
