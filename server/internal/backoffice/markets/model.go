package markets

const (
	DefaultPageLimit int32 = 50
	MaxPageLimit     int32 = 100
)

type Market struct {
	Code      string `json:"code"`
	Currency  string `json:"currency"`
	MinorUnit int16  `json:"minor_unit"`
	IsEnabled bool   `json:"is_enabled"`
}

type ListInput struct {
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

type Page struct {
	Data    []Market `json:"data"`
	Limit   int32    `json:"limit"`
	Offset  int32    `json:"offset"`
	HasMore bool     `json:"has_more"`
}

type CreateInput struct {
	Code      string `json:"code"`
	Currency  string `json:"currency"`
	IsEnabled *bool  `json:"is_enabled,omitempty"`
}

type UpdateInput struct {
	IsEnabled *bool  `json:"is_enabled"`
	Reason    string `json:"reason,omitempty"`
}
