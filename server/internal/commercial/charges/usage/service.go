package usage

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dugble/dugble/server/internal/messaging/email/systemmail"
)

type chargeRepository interface {
	ChargeSMS(context.Context, pgx.Tx, SMSChargeInput) (Charge, error)
	ChargeEmail(context.Context, pgx.Tx, EmailChargeInput) (Charge, error)
}

type balanceNotifier interface {
	SendWalletBalanceAlert(context.Context, systemmail.SendWalletBalanceAlertInput) error
}

type balanceRecipientRepository interface {
	ListBalanceRecipients(context.Context, uuid.UUID) ([]BalanceRecipient, error)
}

type Service struct {
	repository      chargeRepository
	recipients      balanceRecipientRepository
	balanceNotifier balanceNotifier
}

func (s *Service) WithBalanceNotifier(notifier balanceNotifier) *Service {
	s.balanceNotifier = notifier
	return s
}

func NewService(repository chargeRepository) *Service {
	service := &Service{repository: repository}
	service.recipients, _ = repository.(balanceRecipientRepository)
	return service
}

func (s *Service) ChargeEmail(
	ctx context.Context,
	tx pgx.Tx,
	input EmailChargeInput,
) (Charge, error) {
	if err := validateEmailCharge(input); err != nil {
		return Charge{}, err
	}
	result, err := s.repository.ChargeEmail(ctx, tx, input)
	if err != nil {
		return Charge{}, err
	}
	if err := validateEmailChargeResult(result, input.RecipientCount); err != nil {
		return Charge{}, fmt.Errorf("charge email billing: %w", err)
	}
	return result, nil
}

func (s *Service) ChargeSMS(
	ctx context.Context,
	tx pgx.Tx,
	input SMSChargeInput,
) (Charge, error) {
	input, err := validateSMSCharge(input)
	if err != nil {
		return Charge{}, err
	}
	result, err := s.repository.ChargeSMS(ctx, tx, input)
	if err != nil {
		return Charge{}, err
	}
	if err := validateSMSChargeResult(result, input.destinationCountry); err != nil {
		return Charge{}, fmt.Errorf("charge SMS billing: %w", err)
	}
	return result, nil
}
