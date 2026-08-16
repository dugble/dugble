package user

import "time"

// User represents a user in the system.
type User struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"email_verified"`
	Name          string    `json:"name"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type UpdateProfileRequest struct {
	Name string `json:"name"`
}

type UpdateEmailRequest struct {
	Email           string `json:"email"`
	CurrentPassword string `json:"current_password"`
}

type VerifyEmailChangeRequest struct {
	Token string `json:"token"`
}

type EmailChangePending struct {
	Email                 string    `json:"email"`
	PendingEmail          string    `json:"pending_email"`
	VerificationExpiresAt time.Time `json:"verification_expires_at"`
}

type emailChangeRequest struct {
	UserID       string
	PendingEmail string
	RequestedAt  time.Time
	ExpiresAt    time.Time
}

type UpdatePasswordRequest struct {
	Password string `json:"password"`
}
