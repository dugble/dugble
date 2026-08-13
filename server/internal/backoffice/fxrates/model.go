package fxrates

import "time"

type FXRate struct {
	ID             string     `json:"id"`
	BaseCurrency   string     `json:"base_currency"`
	QuoteCurrency  string     `json:"quote_currency"`
	Rate           string     `json:"rate"`
	EffectiveFrom  time.Time  `json:"effective_from"`
	EffectiveUntil *time.Time `json:"effective_until,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type ListInput struct{ Limit, Offset int32 }

type Page struct {
	Data    []FXRate `json:"data"`
	Limit   int32    `json:"limit"`
	Offset  int32    `json:"offset"`
	HasMore bool     `json:"has_more"`
}

type CreateInput struct {
	BaseCurrency   string     `json:"base_currency"`
	QuoteCurrency  string     `json:"quote_currency"`
	Rate           string     `json:"rate"`
	EffectiveFrom  time.Time  `json:"effective_from"`
	EffectiveUntil *time.Time `json:"effective_until,omitempty"`
}

type ReplaceInput struct {
	BaseCurrency  string    `json:"base_currency"`
	QuoteCurrency string    `json:"quote_currency"`
	Rate          string    `json:"rate"`
	EffectiveFrom time.Time `json:"effective_from"`
	Reason        string    `json:"reason"`
}

type CloseInput struct {
	EffectiveUntil time.Time `json:"effective_until"`
	Reason         string    `json:"reason"`
}
