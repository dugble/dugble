package payment

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	sentrymonitoring "github.com/dugble/dugble/server/internal/integrations/monitoring/sentry"
	"github.com/dugble/dugble/server/internal/messaging/email/systemmail"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

type store interface {
	Create(context.Context, uuid.UUID, CreateInput) (Transaction, error)
	GetByClientReference(context.Context, string, string) (Transaction, error)
	MarkPaidAndCredit(context.Context, Transaction, string) (Transaction, error)
	MarkFailed(context.Context, Transaction) (Transaction, error)
	ListRecipients(context.Context, uuid.UUID) ([]Recipient, error)
}

type notifier interface {
	SendWalletTopUpResult(context.Context, systemmail.SendWalletTopUpResultInput) error
}

type Service struct {
	repository store
	notifier   notifier
}

func NewService(repository store) *Service { return &Service{repository: repository} }

func (s *Service) WithNotifier(notifier notifier) *Service {
	s.notifier = notifier
	return s
}

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
	s.notify(ctx, transaction)
	return transaction, nil
}

func (s *Service) Fail(ctx context.Context, input FailInput) (Transaction, error) {
	input.Provider = strings.TrimSpace(input.Provider)
	input.ClientReference = strings.TrimSpace(input.ClientReference)
	if input.Provider == "" || input.ClientReference == "" {
		return Transaction{}, apperrors.NewBadRequest("Payment provider and client reference are required")
	}
	transaction, err := s.repository.GetByClientReference(ctx, input.Provider, input.ClientReference)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Transaction{}, apperrors.NewNotFound("Payment transaction not found")
		}
		return Transaction{}, apperrors.NewInternal("Unable to get payment", err)
	}
	if transaction.Status == StatusFailed {
		return transaction, nil
	}
	if transaction.Status != StatusPending {
		return Transaction{}, apperrors.NewConflict("Payment transaction cannot be failed")
	}
	transaction, err = s.repository.MarkFailed(ctx, transaction)
	if err != nil {
		return Transaction{}, apperrors.NewInternal("Unable to fail payment", err)
	}
	s.notify(ctx, transaction)
	return transaction, nil
}

func (s *Service) notify(ctx context.Context, transaction Transaction) {
	if s.notifier == nil {
		return
	}
	teamID, err := uuid.Parse(transaction.TeamID)
	if err != nil {
		sentrymonitoring.Warn("failed to resolve wallet top-up recipients", "team_id", transaction.TeamID, "error", err)
		return
	}
	recipients, err := s.repository.ListRecipients(ctx, teamID)
	if err != nil {
		sentrymonitoring.Warn("failed to resolve wallet top-up recipients", "team_id", transaction.TeamID, "error", err)
		return
	}
	for _, recipient := range recipients {
		input := systemmail.SendWalletTopUpResultInput{
			ToEmail: recipient.Email, Name: recipient.Name, TeamName: recipient.TeamName,
			Currency: transaction.Currency, AmountUnits: transaction.AmountUnits,
			ClientReference: transaction.ClientReference, Status: transaction.Status,
		}
		if err := s.notifier.SendWalletTopUpResult(ctx, input); err != nil {
			sentrymonitoring.Warn("failed to send wallet top-up notification", "team_id", transaction.TeamID, "status", transaction.Status, "error", err)
			return
		}
	}
}
