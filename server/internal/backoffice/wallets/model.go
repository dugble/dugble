package wallets

import "time"

type Wallet struct {
	TeamID        string    `json:"team_id"`
	TeamName      string    `json:"team_name"`
	BillingMarket string    `json:"billing_market"`
	Currency      string    `json:"currency"`
	BalanceUnits  int64     `json:"balance_units"`
	Tier          string    `json:"tier"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Transaction struct {
	ID                   string    `json:"id"`
	TeamID               string    `json:"team_id"`
	TeamName             string    `json:"team_name"`
	UsageAuthorizationID *string   `json:"usage_authorization_id,omitempty"`
	SubscriptionChargeID *string   `json:"subscription_charge_id,omitempty"`
	PaymentTransactionID *string   `json:"payment_transaction_id,omitempty"`
	AmountUnits          int64     `json:"amount_units"`
	Currency             string    `json:"currency"`
	TransactionType      string    `json:"transaction_type"`
	ReferenceID          string    `json:"reference_id"`
	CreatedAt            time.Time `json:"created_at"`
}

type ListInput struct{ Limit, Offset int32 }
type TransactionListInput struct {
	TeamID string
	Limit  int32
	Offset int32
}
type WalletPage struct {
	Data    []Wallet `json:"data"`
	Limit   int32    `json:"limit"`
	Offset  int32    `json:"offset"`
	HasMore bool     `json:"has_more"`
}
type TransactionPage struct {
	Data    []Transaction `json:"data"`
	Limit   int32         `json:"limit"`
	Offset  int32         `json:"offset"`
	HasMore bool          `json:"has_more"`
}
type AdjustmentInput struct {
	AmountUnits int64  `json:"amount_units"`
	ReferenceID string `json:"reference_id"`
	Reason      string `json:"reason"`
	ActorUserID string `json:"-"`
	SessionID   string `json:"-"`
}
