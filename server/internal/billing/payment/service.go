package payment

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apperrors "github.com/coffeyvidzro/dugble/server/pkg/errors"
)

type store interface {
	Create(context.Context, uuid.UUID, CreateInput) (Transaction, error)
	GetByClientReference(context.Context, string, string) (Transaction, error)
	MarkPaidAndCredit(context.Context, Transaction, string) (Transaction, error)
}

type Service struct{ repository store }

func NewService(repository store) *Service { return &Service{repository: repository} }

func (s *Service) Create(ctx context.Context, input CreateInput) (Transaction, error) {
	teamID, input, err := validateCreate(input)
	if err != nil {
		return Transaction{}, err
	}
	transaction, err := s.repository.Create(ctx, teamID, input)
	if err != nil {
		return Transaction{}, apperrors.NewInternal("Unable to create payment", err)
	}
	return transaction, nil
}

func (s *Service) Complete(ctx context.Context, input CompleteInput) (Transaction, error) {
	input, err := validateComplete(input)
	if err != nil {
		return Transaction{}, err
	}
	transaction, err := s.repository.GetByClientReference(ctx, input.Provider, input.ClientReference)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Transaction{}, apperrors.NewNotFound("Payment transaction not found")
		}
		return Transaction{}, apperrors.NewInternal("Unable to get payment", err)
	}
	if transaction.AmountUnits != input.AmountUnits {
		return Transaction{}, apperrors.NewBadRequest("Payment amount does not match transaction")
	}
	if transaction.Status == StatusPaid {
		if transaction.ProviderTransactionID != nil && *transaction.ProviderTransactionID == input.ProviderTransactionID {
			return transaction, nil
		}
		return Transaction{}, apperrors.NewConflict("Payment transaction is already completed")
	}
	if transaction.Status != StatusPending {
		return Transaction{}, apperrors.NewConflict("Payment transaction cannot be completed")
	}
	transaction, err = s.repository.MarkPaidAndCredit(ctx, transaction, input.ProviderTransactionID)
	if err != nil {
		return Transaction{}, apperrors.NewInternal("Unable to complete payment", err)
	}
	return transaction, nil
}
