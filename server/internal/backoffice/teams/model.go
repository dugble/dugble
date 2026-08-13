package teams

import "time"

const (
	DefaultPageLimit int32 = 50
	MaxPageLimit     int32 = 100
)

type Filter struct {
	Query  string `json:"q,omitempty"`
	Status string `json:"status,omitempty"`
	Limit  int32  `json:"limit"`
	Offset int32  `json:"offset"`
}

type Page struct {
	Data    []Row `json:"data"`
	Limit   int32 `json:"limit"`
	Offset  int32 `json:"offset"`
	HasMore bool  `json:"has_more"`
}

type Row struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Detail struct {
	Team    Row         `json:"team"`
	Members []MemberRow `json:"members"`
	SMS     []SMSRow    `json:"recent_sms"`
}

type MemberRow struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type SMSRow struct {
	ID           string    `json:"id"`
	TeamName     string    `json:"team_name"`
	ToNumber     string    `json:"to_number"`
	FromName     string    `json:"from_name"`
	Status       string    `json:"status"`
	ProviderID   string    `json:"provider_id,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type StatusRequest struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}
