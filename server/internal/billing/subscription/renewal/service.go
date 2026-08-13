package renewal

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	charges "github.com/coffeyvidzro/dugble/server/internal/billing/charge/subscription"
	"github.com/coffeyvidzro/dugble/server/internal/billing/subscription/lifecycle"
)

type renewalStore interface {
	GetDue(context.Context, pgx.Tx, uuid.UUID) (Due, error)
	ApplyCancellation(context.Context, pgx.Tx, uuid.UUID) (timeRange, error)
	ApplyCharge(context.Context, pgx.Tx, uuid.UUID, bool, string, time.Time, time.Time) (appliedState, error)
}
type periodCharger interface {
	ChargePeriod(context.Context, pgx.Tx, charges.Input) (charges.Result, error)
}
type decider interface {
	Decide(lifecycle.State, time.Time) (lifecycle.Decision, error)
}
type eventPublisher interface {
	PublishTx(context.Context, pgx.Tx, Result) error
}

type Service struct {
	repository renewalStore
	charger    periodCharger
	lifecycle  decider
	events     eventPublisher
	now        func() time.Time
}

func (s *Service) WithEventPublisher(publisher eventPublisher) *Service {
	s.events = publisher
	return s
}

func NewService(repository renewalStore, charger periodCharger, lifecycleService decider) *Service {
	return &Service{repository: repository, charger: charger, lifecycle: lifecycleService, now: time.Now}
}

func (s *Service) ProcessTeam(ctx context.Context, tx pgx.Tx, teamID uuid.UUID) (Result, error) {
	due, err := s.repository.GetDue(ctx, tx, teamID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Result{TeamID: teamID, Outcome: OutcomeNotDue}, nil
	}
	if err != nil {
		return Result{}, err
	}
	decision, err := s.lifecycle.Decide(due.State, s.now())
	if err != nil {
		return Result{}, err
	}
	base := Result{SubscriptionID: due.SubscriptionID, TeamID: due.TeamID, PreviousPlan: due.State.PlanCode, CurrentPlan: due.State.PlanCode, PeriodStart: due.State.CurrentPeriodStart, PeriodEnd: due.State.CurrentPeriodEnd}
	if decision.Transition == lifecycle.TransitionCancel {
		period, err := s.repository.ApplyCancellation(ctx, tx, due.SubscriptionID)
		if err != nil {
			return Result{}, err
		}
		base.Outcome, base.PeriodStart, base.PeriodEnd = OutcomeCanceled, period.Start, period.End
		if err := s.publish(ctx, tx, base); err != nil {
			return Result{}, err
		}
		return base, nil
	}
	charge, err := s.charger.ChargePeriod(ctx, tx, charges.Input{SubscriptionID: due.SubscriptionID, TeamID: due.TeamID, PlanCode: decision.NextPlan, PeriodStart: decision.PeriodStart, PeriodEnd: decision.PeriodEnd})
	if err != nil {
		return Result{}, err
	}
	applied := charge.Outcome == charges.OutcomeApplied || charge.Outcome == charges.OutcomeAlreadyApplied
	state, err := s.repository.ApplyCharge(ctx, tx, due.SubscriptionID, applied, decision.NextPlan, decision.PeriodStart, decision.PeriodEnd)
	if err != nil {
		return Result{}, err
	}
	base.CurrentPlan, base.PeriodStart, base.PeriodEnd = state.Plan, state.Period.Start, state.Period.End
	base.Charge = charge
	switch {
	case charge.Outcome == charges.OutcomePriceUnavailable:
		base.Outcome = OutcomePriceUnavailable
	case !applied:
		base.Outcome = OutcomePastDue
	case decision.Transition == lifecycle.TransitionChangePlan:
		base.Outcome = OutcomePlanChanged
	default:
		base.Outcome = OutcomeRenewed
	}
	if err := s.publish(ctx, tx, base); err != nil {
		return Result{}, err
	}
	return base, nil
}

func (s *Service) publish(ctx context.Context, tx pgx.Tx, result Result) error {
	if s.events == nil || result.Outcome == OutcomeNotDue {
		return nil
	}
	return s.events.PublishTx(ctx, tx, result)
}
