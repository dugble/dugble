package senderid

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	StatusPending   = "pending"
	StatusApproved  = "approved"
	StatusRejected  = "rejected"
	StatusSuspended = "suspended"
	StatusInactive  = "inactive"

	providerStatusSubmissionFailed  = "submission_failed"
	providerStatusSubmissionUnknown = "submission_unknown"
)

var (
	ErrSenderIDAlreadyExists = errors.New("sender id already exists")
	ErrJobNotConfigured      = errors.New("sender ID reconciliation job is not configured")
	ErrInvalidJobConfig      = errors.New("invalid Sender ID reconciliation job configuration")
	ErrRegistrationClaimLost = errors.New("sender ID registration claim was lost")
	ErrWorkerIDRequired      = errors.New("sender ID reconciliation worker ID is required")
)

type SenderID struct {
	ID              string     `json:"id"`
	TeamID          string     `json:"team_id"`
	Name            string     `json:"name"`
	CountryCode     string     `json:"country_code"`
	Purpose         string     `json:"purpose"`
	Status          string     `json:"status"`
	Provider        *string    `json:"-"`
	RejectionReason *string    `json:"rejection_reason,omitempty"`
	ApprovedAt      *time.Time `json:"approved_at,omitempty"`
	RejectedAt      *time.Time `json:"rejected_at,omitempty"`
	SuspendedAt     *time.Time `json:"suspended_at,omitempty"`
	CreatedBy       *string    `json:"created_by,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type CreateRequest struct {
	Name        string  `json:"name"`
	CountryCode string  `json:"country_code"`
	Purpose     string  `json:"purpose"`
	Provider    *string `json:"provider,omitempty"`
}

type RegistrationClaim struct {
	ID                  uuid.UUID
	TeamID              uuid.UUID
	Name                string
	CountryCode         string
	Provider            string
	ProviderStatus      string
	ProviderSubmittedAt *time.Time
	Attempt             int32
}

type safeFallbackError interface {
	error
	SafeToFallback() bool
}
