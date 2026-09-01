package payment

import "time"

const (
	ProviderHubtel = "hubtel"
	CurrencyGHS    = "GHS"
	StatusPending  = "pending"
	StatusPaid     = "paid"
	StatusFailed   = "failed"
)

type Transaction struct {
	ID                    string     `json:"id"`
	TeamID                string     `json:"team_id"`
	Provider              string     `json:"provider"`
	ClientReference       string     `json:"client_reference"`
	Currency              string     `json:"currency"`
	AmountUnits           int64      `json:"amount_units"`
	Status                string     `json:"status"`
	ProviderTransactionID *string    `json:"provider_transaction_id,omitempty"`
	PaidAt                *time.Time `json:"paid_at,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

type CreateInput struct {
	TeamID          string
	Provider        string
	ClientReference string
	Currency        string
	AmountUnits     int64
}

type CompleteInput struct {
	Provider              string
	ClientReference       string
	ProviderTransactionID string
	AmountUnits           int64
}

type FailInput struct {
	Provider        string
	ClientReference string
}

type Recipient struct {
	Name     string
	Email    string
	TeamName string
}
