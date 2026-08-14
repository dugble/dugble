package payment

import (
	"strings"

	"github.com/google/uuid"

	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

func validateCreate(input CreateInput) (uuid.UUID, CreateInput, error) {
	teamID, err := uuid.Parse(strings.TrimSpace(input.TeamID))
	if err != nil {
		return uuid.Nil, CreateInput{}, apperrors.NewBadRequest("Team id must be a valid UUID")
	}
	input.Provider = strings.ToLower(strings.TrimSpace(input.Provider))
	input.ClientReference = strings.TrimSpace(input.ClientReference)
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	if input.Provider == "" || strings.ContainsAny(input.Provider, " \t\r\n") {
		return uuid.Nil, CreateInput{}, apperrors.NewBadRequest("Payment provider is invalid")
	}
	if input.ClientReference == "" {
		return uuid.Nil, CreateInput{}, apperrors.NewBadRequest("Payment client reference is required")
	}
	if !validCurrencyCode(input.Currency) {
		return uuid.Nil, CreateInput{}, apperrors.NewBadRequest("Payment currency is invalid")
	}
	if input.AmountUnits <= 0 {
		return uuid.Nil, CreateInput{}, apperrors.NewBadRequest("Payment amount must be greater than zero")
	}
	return teamID, input, nil
}

func validateComplete(input CompleteInput) (CompleteInput, error) {
	input.Provider = strings.ToLower(strings.TrimSpace(input.Provider))
	input.ClientReference = strings.TrimSpace(input.ClientReference)
	input.ProviderTransactionID = strings.TrimSpace(input.ProviderTransactionID)
	if input.Provider == "" || strings.ContainsAny(input.Provider, " \t\r\n") {
		return CompleteInput{}, apperrors.NewBadRequest("Payment provider is invalid")
	}
	if input.ClientReference == "" {
		return CompleteInput{}, apperrors.NewBadRequest("Payment client reference is required")
	}
	if input.ProviderTransactionID == "" {
		return CompleteInput{}, apperrors.NewBadRequest("Provider transaction id is required")
	}
	if input.AmountUnits <= 0 {
		return CompleteInput{}, apperrors.NewBadRequest("Payment amount must be greater than zero")
	}
	return input, nil
}

func validCurrencyCode(value string) bool {
	if len(value) != 3 {
		return false
	}
	for _, character := range value {
		if character < 'A' || character > 'Z' {
			return false
		}
	}
	return true
}
