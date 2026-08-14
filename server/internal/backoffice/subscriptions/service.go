package subscriptions

import (
	"context"
	"errors"
	"strings"

	apperrors "github.com/dugble/dugble/server/pkg/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Service struct{ repository *Repository }

func NewService(r *Repository) *Service { return &Service{r} }
func (s *Service) List(ctx context.Context, f Filter) (Page, error) {
	f.Status = strings.ToLower(strings.TrimSpace(f.Status))
	if f.Status != "" && f.Status != "active" && f.Status != "past_due" && f.Status != "canceled" {
		return Page{}, apperrors.NewBadRequest("status must be active, past_due, or canceled")
	}
	limit, offset, e := page(f.Limit, f.Offset)
	if e != nil {
		return Page{}, e
	}
	var teamID *uuid.UUID
	if strings.TrimSpace(f.TeamID) != "" {
		v, x := uuid.Parse(strings.TrimSpace(f.TeamID))
		if x != nil {
			return Page{}, apperrors.NewBadRequest("Invalid team ID")
		}
		teamID = &v
	}
	f.Limit, f.Offset = limit+1, offset
	items, e := s.repository.List(ctx, f, teamID)
	if e != nil {
		return Page{}, apperrors.NewInternal("Unable to list subscriptions", e)
	}
	more := len(items) > int(limit)
	if more {
		items = items[:limit]
	}
	return Page{items, limit, offset, more}, nil
}
func (s *Service) Get(ctx context.Context, value string) (Subscription, error) {
	id, e := uuid.Parse(strings.TrimSpace(value))
	if e != nil {
		return Subscription{}, apperrors.NewBadRequest("Invalid subscription ID")
	}
	item, e := s.repository.Get(ctx, id)
	if errors.Is(e, pgx.ErrNoRows) {
		return Subscription{}, apperrors.NewNotFound("Subscription not found")
	}
	if e != nil {
		return Subscription{}, apperrors.NewInternal("Unable to get subscription", e)
	}
	return item, nil
}
func (s *Service) Charges(ctx context.Context, value string, limit, offset int32) (ChargePage, error) {
	id, e := uuid.Parse(strings.TrimSpace(value))
	if e != nil {
		return ChargePage{}, apperrors.NewBadRequest("Invalid subscription ID")
	}
	limit, offset, e = page(limit, offset)
	if e != nil {
		return ChargePage{}, e
	}
	items, e := s.repository.Charges(ctx, id, limit+1, offset)
	if e != nil {
		return ChargePage{}, apperrors.NewInternal("Unable to list subscription charges", e)
	}
	more := len(items) > int(limit)
	if more {
		items = items[:limit]
	}
	return ChargePage{items, limit, offset, more}, nil
}

func (s *Service) ChangePlan(ctx context.Context, value string, input ChangePlanInput) (Subscription, error) {
	plan := strings.ToLower(strings.TrimSpace(input.PlanCode))
	if plan == "" {
		return Subscription{}, apperrors.NewBadRequest("Plan code is required")
	}
	item, subscriptionID, teamID, actorID, reason, err := s.actionValues(ctx, value, input.Reason, input.ActorUserID)
	if err != nil {
		return Subscription{}, err
	}
	if err := s.repository.ChangePlan(ctx, subscriptionID, teamID, plan, reason, actorID, input.SessionID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Subscription{}, apperrors.NewConflict("Plan change is not valid for the subscription")
		}
		return Subscription{}, apperrors.NewInternal("Unable to schedule plan change", err)
	}
	return s.Get(ctx, item.ID)
}
func (s *Service) CancelPlanChange(ctx context.Context, value string, input ActionInput) (Subscription, error) {
	return s.lifecycle(ctx, value, input, "cancel pending plan change", (*Repository).CancelPlanChange)
}
func (s *Service) Cancel(ctx context.Context, value string, input ActionInput) (Subscription, error) {
	return s.lifecycle(ctx, value, input, "cancel subscription", (*Repository).Cancel)
}
func (s *Service) Reactivate(ctx context.Context, value string, input ActionInput) (Subscription, error) {
	return s.lifecycle(ctx, value, input, "reactivate subscription", (*Repository).Reactivate)
}

type lifecycleOperation func(*Repository, context.Context, uuid.UUID, uuid.UUID, string, uuid.UUID, string) error

func (s *Service) lifecycle(ctx context.Context, value string, input ActionInput, label string, operation lifecycleOperation) (Subscription, error) {
	item, subscriptionID, teamID, actorID, reason, err := s.actionValues(ctx, value, input.Reason, input.ActorUserID)
	if err != nil {
		return Subscription{}, err
	}
	if err := operation(s.repository, ctx, subscriptionID, teamID, reason, actorID, input.SessionID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Subscription{}, apperrors.NewConflict("Unable to " + label + " in its current state")
		}
		return Subscription{}, apperrors.NewInternal("Unable to "+label, err)
	}
	return s.Get(ctx, item.ID)
}
func (s *Service) actionValues(ctx context.Context, value, reason, actorValue string) (Subscription, uuid.UUID, uuid.UUID, uuid.UUID, string, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return Subscription{}, uuid.Nil, uuid.Nil, uuid.Nil, "", apperrors.NewBadRequest("Reason is required")
	}
	actorID, err := uuid.Parse(actorValue)
	if err != nil {
		return Subscription{}, uuid.Nil, uuid.Nil, uuid.Nil, "", apperrors.NewBadRequest("Authenticated administrator is invalid")
	}
	item, err := s.Get(ctx, value)
	if err != nil {
		return Subscription{}, uuid.Nil, uuid.Nil, uuid.Nil, "", err
	}
	subscriptionID, _ := uuid.Parse(item.ID)
	teamID, _ := uuid.Parse(item.TeamID)
	return item, subscriptionID, teamID, actorID, reason, nil
}
func page(limit, offset int32) (int32, int32, error) {
	if limit < 0 || limit > 100 {
		return 0, 0, apperrors.NewBadRequest("Limit must be between 1 and 100")
	}
	if offset < 0 {
		return 0, 0, apperrors.NewBadRequest("Offset must not be negative")
	}
	if limit == 0 {
		limit = 50
	}
	return limit, offset, nil
}
