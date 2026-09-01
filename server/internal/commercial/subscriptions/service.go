package subscription

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dugble/dugble/server/internal/messaging/email/systemmail"
	"github.com/dugble/dugble/server/internal/security/authz"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

type store interface {
	GetSubscription(context.Context, uuid.UUID) (Subscription, error)
	ListCharges(context.Context, uuid.UUID, int32, int32) ([]Charge, error)
	SchedulePlanChange(context.Context, uuid.UUID, string) (Subscription, error)
	CancelPlanChange(context.Context, uuid.UUID) (Subscription, error)
	Cancel(context.Context, uuid.UUID) (Subscription, error)
	Reactivate(context.Context, uuid.UUID) (Subscription, error)
	ListNotificationRecipients(context.Context, uuid.UUID) ([]systemmail.Recipient, error)
}
type notifier interface {
	SendSubscriptionChange(context.Context, systemmail.SendSubscriptionChangeInput) error
}
type Service struct {
	repository store
	notifier   notifier
}

func NewService(repository store) *Service                 { return &Service{repository: repository} }
func (s *Service) WithNotifier(notifier notifier) *Service { s.notifier = notifier; return s }
func (s *Service) Get(ctx context.Context) (Subscription, error) {
	access, decision := authz.ResolveAccess(ctx, authz.PermissionWalletRead)
	if !decision.Allowed {
		return Subscription{}, apperrors.NewForbidden(decision.Reason)
	}
	result, err := s.repository.GetSubscription(ctx, access.Scope.TeamID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, apperrors.NewNotFound("Subscription not found")
	}
	if err != nil {
		return Subscription{}, apperrors.NewInternal("Unable to get subscription", err)
	}
	return result, nil
}

func (s *Service) ListCharges(ctx context.Context, limit, offset int32) (ChargePage, error) {
	access, decision := authz.ResolveAccess(ctx, authz.PermissionWalletRead)
	if !decision.Allowed {
		return ChargePage{}, apperrors.NewForbidden(decision.Reason)
	}
	if limit == 0 {
		limit = 50
	}
	if limit < 0 || limit > 100 {
		return ChargePage{}, apperrors.NewBadRequest("Subscription charge limit must be between 1 and 100")
	}
	if offset < 0 {
		return ChargePage{}, apperrors.NewBadRequest("Subscription charge offset cannot be negative")
	}
	charges, err := s.repository.ListCharges(ctx, access.Scope.TeamID, limit, offset)
	if err != nil {
		return ChargePage{}, apperrors.NewInternal("Unable to list subscription charges", err)
	}
	return ChargePage{Charges: charges, Limit: limit, Offset: offset}, nil
}
func (s *Service) SelectPlan(ctx context.Context, input SelectPlanInput) (Subscription, error) {
	access, decision := authz.ResolveAccess(ctx, authz.PermissionTeamUpdate)
	if !decision.Allowed {
		return Subscription{}, apperrors.NewForbidden(decision.Reason)
	}
	plan := strings.ToLower(strings.TrimSpace(input.Plan))
	if !validPlan(plan) {
		return Subscription{}, apperrors.NewBadRequest("Plan must be growth, scale, or enterprise")
	}
	result, err := s.repository.SchedulePlanChange(ctx, access.Scope.TeamID, plan)
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, apperrors.NewConflict("Plan is unavailable for the next billing period")
	}
	if err != nil {
		return Subscription{}, apperrors.NewInternal("Unable to schedule plan change", err)
	}
	s.notify(ctx, access.Scope.TeamID, systemmail.SendSubscriptionChangeInput{CurrentPlan: result.PlanCode, NewPlan: plan, Event: "plan_change_scheduled", EffectiveAt: formatEffectiveAt(result.PendingPlanEffectiveAt)})
	return result, nil
}
func (s *Service) CancelPendingPlanChange(ctx context.Context) (Subscription, error) {
	access, decision := authz.ResolveAccess(ctx, authz.PermissionTeamUpdate)
	if !decision.Allowed {
		return Subscription{}, apperrors.NewForbidden(decision.Reason)
	}
	previous, _ := s.repository.GetSubscription(ctx, access.Scope.TeamID)
	result, err := s.repository.CancelPlanChange(ctx, access.Scope.TeamID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, apperrors.NewConflict("Subscription cannot be changed")
	}
	if err != nil {
		return Subscription{}, apperrors.NewInternal("Unable to cancel pending plan change", err)
	}
	newPlan := ""
	if previous.PendingPlanCode != nil {
		newPlan = *previous.PendingPlanCode
	}
	s.notify(ctx, access.Scope.TeamID, systemmail.SendSubscriptionChangeInput{CurrentPlan: result.PlanCode, NewPlan: newPlan, Event: "plan_change_canceled"})
	return result, nil
}
func (s *Service) Cancel(ctx context.Context) (Subscription, error) {
	access, decision := authz.ResolveAccess(ctx, authz.PermissionTeamUpdate)
	if !decision.Allowed {
		return Subscription{}, apperrors.NewForbidden(decision.Reason)
	}
	result, err := s.repository.Cancel(ctx, access.Scope.TeamID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, apperrors.NewConflict("Subscription cannot be canceled")
	}
	if err != nil {
		return Subscription{}, apperrors.NewInternal("Unable to cancel subscription", err)
	}
	s.notify(ctx, access.Scope.TeamID, systemmail.SendSubscriptionChangeInput{CurrentPlan: result.PlanCode, Event: "cancellation_scheduled", EffectiveAt: result.CurrentPeriodEnd.UTC().Format("2 January 2006")})
	return result, nil
}

func (s *Service) Reactivate(ctx context.Context) (Subscription, error) {
	access, decision := authz.ResolveAccess(ctx, authz.PermissionTeamUpdate)
	if !decision.Allowed {
		return Subscription{}, apperrors.NewForbidden(decision.Reason)
	}
	result, err := s.repository.Reactivate(ctx, access.Scope.TeamID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, apperrors.NewConflict("Subscription cannot be reactivated")
	}
	if err != nil {
		return Subscription{}, apperrors.NewInternal("Unable to reactivate subscription", err)
	}
	s.notify(ctx, access.Scope.TeamID, systemmail.SendSubscriptionChangeInput{CurrentPlan: result.PlanCode, Event: "cancellation_reversed"})
	return result, nil
}

func (s *Service) notify(ctx context.Context, teamID uuid.UUID, input systemmail.SendSubscriptionChangeInput) {
	if s.notifier == nil {
		return
	}
	recipients, err := s.repository.ListNotificationRecipients(ctx, teamID)
	if err != nil {
		return
	}
	for _, recipient := range recipients {
		input.ToEmail, input.Name = recipient.Email, recipient.Name
		_ = s.notifier.SendSubscriptionChange(ctx, input)
	}
}

func formatEffectiveAt(value *time.Time) string {
	if value == nil {
		return "the next billing period"
	}
	return value.UTC().Format("2 January 2006")
}
