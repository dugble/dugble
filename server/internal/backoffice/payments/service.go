package payments

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
	f.Provider = strings.ToLower(strings.TrimSpace(f.Provider))
	if f.Status != "" && f.Status != "pending" && f.Status != "paid" && f.Status != "failed" {
		return Page{}, apperrors.NewBadRequest("status must be pending, paid, or failed")
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
		return Page{}, apperrors.NewInternal("Unable to list payments", e)
	}
	more := len(items) > int(limit)
	if more {
		items = items[:limit]
	}
	return Page{items, limit, offset, more}, nil
}
func (s *Service) Get(ctx context.Context, value string) (Payment, error) {
	id, e := uuid.Parse(strings.TrimSpace(value))
	if e != nil {
		return Payment{}, apperrors.NewBadRequest("Invalid payment ID")
	}
	item, e := s.repository.Get(ctx, id)
	if errors.Is(e, pgx.ErrNoRows) {
		return Payment{}, apperrors.NewNotFound("Payment not found")
	}
	if e != nil {
		return Payment{}, apperrors.NewInternal("Unable to get payment", e)
	}
	return item, nil
}

func (s *Service) Reconcile(ctx context.Context, value string, input ReconcileInput) (Payment, error) {
	input.ProviderTransactionID = strings.TrimSpace(input.ProviderTransactionID)
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	input.Reason = strings.TrimSpace(input.Reason)
	if input.ProviderTransactionID == "" {
		return Payment{}, apperrors.NewBadRequest("Provider transaction ID is required")
	}
	if input.AmountUnits <= 0 {
		return Payment{}, apperrors.NewBadRequest("Amount units must be positive")
	}
	if input.Currency == "" {
		return Payment{}, apperrors.NewBadRequest("Currency is required")
	}
	if input.Reason == "" {
		return Payment{}, apperrors.NewBadRequest("Reason is required")
	}
	actorID, err := uuid.Parse(input.ActorUserID)
	if err != nil {
		return Payment{}, apperrors.NewBadRequest("Authenticated administrator is invalid")
	}
	item, err := s.Get(ctx, value)
	if err != nil {
		return Payment{}, err
	}
	if input.AmountUnits != item.AmountUnits || input.Currency != item.Currency {
		return Payment{}, apperrors.NewConflict("Reconciliation amount or currency does not match payment")
	}
	if item.Status == "paid" {
		if item.ProviderTransactionID != nil && *item.ProviderTransactionID == input.ProviderTransactionID {
			return item, nil
		}
		return Payment{}, apperrors.NewConflict("Payment is already paid")
	}
	if item.Status != "pending" {
		return Payment{}, apperrors.NewConflict("Only pending payments can be reconciled")
	}
	if err := s.repository.Reconcile(ctx, item, input.ProviderTransactionID, input.Reason, actorID, input.SessionID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Payment{}, apperrors.NewConflict("Payment changed while it was being reconciled")
		}
		return Payment{}, apperrors.NewInternal("Unable to reconcile payment", err)
	}
	return s.Get(ctx, value)
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
