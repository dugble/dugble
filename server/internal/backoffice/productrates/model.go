package productrates

import "time"

type ProductRate struct {
	ID             string     `json:"id"`
	Product        string     `json:"product"`
	Meter          string     `json:"meter"`
	BillingMarket  string     `json:"billing_market"`
	Tier           string     `json:"tier"`
	Currency       string     `json:"currency"`
	CostUnits      int64      `json:"cost_units"`
	EffectiveFrom  time.Time  `json:"effective_from"`
	EffectiveUntil *time.Time `json:"effective_until,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type ListInput struct {
	Limit  int32
	Offset int32
}

type Page struct {
	Data    []ProductRate `json:"data"`
	Limit   int32         `json:"limit"`
	Offset  int32         `json:"offset"`
	HasMore bool          `json:"has_more"`
}

type CreateInput struct {
	Product        string     `json:"product"`
	Meter          string     `json:"meter"`
	BillingMarket  string     `json:"billing_market"`
	Tier           string     `json:"tier"`
	Currency       string     `json:"currency"`
	CostUnits      int64      `json:"cost_units"`
	EffectiveFrom  time.Time  `json:"effective_from"`
	EffectiveUntil *time.Time `json:"effective_until,omitempty"`
}

type CloseInput struct {
	EffectiveUntil time.Time `json:"effective_until"`
	Reason         string    `json:"reason"`
}

type ReplaceInput struct {
	Replacement CreateInput `json:"replacement"`
	Reason      string      `json:"reason"`
}
