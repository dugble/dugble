package allowances

import "time"

type AllowancePolicy struct {
	ID               string     `json:"id"`
	Product          string     `json:"product"`
	Meter            string     `json:"meter"`
	BillingMarket    string     `json:"billing_market"`
	Tier             string     `json:"tier"`
	IncludedQuantity int64      `json:"included_quantity"`
	Cadence          string     `json:"cadence"`
	EffectiveFrom    time.Time  `json:"effective_from"`
	EffectiveUntil   *time.Time `json:"effective_until,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type ListInput struct {
	Limit  int32
	Offset int32
}

type Page struct {
	Data    []AllowancePolicy `json:"data"`
	Limit   int32             `json:"limit"`
	Offset  int32             `json:"offset"`
	HasMore bool              `json:"has_more"`
}

type CreateInput struct {
	Product          string     `json:"product"`
	Meter            string     `json:"meter"`
	BillingMarket    string     `json:"billing_market"`
	Tier             string     `json:"tier"`
	IncludedQuantity int64      `json:"included_quantity"`
	Cadence          string     `json:"cadence,omitempty"`
	EffectiveFrom    time.Time  `json:"effective_from"`
	EffectiveUntil   *time.Time `json:"effective_until,omitempty"`
}

type CloseInput struct {
	EffectiveUntil time.Time `json:"effective_until"`
	Reason         string    `json:"reason"`
}

type ReplaceInput struct {
	Replacement CreateInput `json:"replacement"`
	Reason      string      `json:"reason"`
}
