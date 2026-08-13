package payments

import "time"

type Payment struct {
	ID                    string     `json:"id"`
	TeamID                string     `json:"team_id"`
	TeamName              string     `json:"team_name"`
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
type Filter struct {
	TeamID, Status, Provider string
	Limit, Offset            int32
}
type Page struct {
	Data    []Payment `json:"data"`
	Limit   int32     `json:"limit"`
	Offset  int32     `json:"offset"`
	HasMore bool      `json:"has_more"`
}

type ReconcileInput struct {
	ProviderTransactionID string `json:"provider_transaction_id"`
	AmountUnits           int64  `json:"amount_units"`
	Currency              string `json:"currency"`
	Reason                string `json:"reason"`
	ActorUserID           string `json:"-"`
	SessionID             string `json:"-"`
}
