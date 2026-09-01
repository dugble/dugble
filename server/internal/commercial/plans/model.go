package plan

import "time"

type Price struct {
	ID          string `json:"id"`
	Currency    string `json:"currency"`
	AmountUnits int64  `json:"amount_units"`
}

type Plan struct {
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Price       *Price    `json:"price,omitempty"`
	Available   bool      `json:"available"`
	Current     bool      `json:"current"`
	Pending     bool      `json:"pending"`
	EffectiveAt time.Time `json:"effective_at"`
}
