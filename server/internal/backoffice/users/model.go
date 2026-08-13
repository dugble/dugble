package users

import "time"

const (
	DefaultPageLimit int32 = 50
	MaxPageLimit     int32 = 100
)

type Filter struct {
	Query  string `json:"q,omitempty"`
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
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
	EmailVerified bool      `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
}

type Detail struct {
	User  Row                 `json:"user"`
	Teams []TeamMembershipRow `json:"teams"`
}

type TeamMembershipRow struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Role   string `json:"role"`
	Status string `json:"status"`
}
