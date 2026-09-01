package domainclaim

import (
	"errors"
	"time"

	platformemail "github.com/dugble/dugble/server/internal/messaging/email/provider"
)

const (
	StatusPending    = "pending"
	StatusVerified   = "verified"
	StatusCompleted  = "completed"
	StatusBlocked    = "blocked"
	StatusExpired    = "expired"
	StatusSuperseded = "superseded"
	StatusCanceled   = "canceled"
	StatusFailed     = "failed"

	BlockedGracePeriod            = "grace_period"
	BlockedRecentOwnerActivity    = "recent_owner_activity"
	BlockedPendingScheduledEmails = "pending_scheduled_emails"

	DefaultClaimLifetime             = 7 * 24 * time.Hour
	DefaultOwnershipGracePeriod      = 72 * time.Hour
	DefaultRecentOwnerActivityWindow = 24 * time.Hour
)

var (
	ErrClaimAlreadyExists = errors.New("domain claim already exists")
	ErrAlreadyOwned       = errors.New("domain is already owned by the team")
	ErrClaimNotReady      = errors.New("domain claim is not ready for completion")
)

type VerificationRecord = platformemail.VerificationRecord

type Request struct {
	Name   string `json:"name"`
	Region string `json:"region"`
	TLS    string `json:"tls,omitempty"`
}

type Claim struct {
	ID                      string             `json:"id"`
	DomainID                string             `json:"domain_id"`
	Name                    string             `json:"name"`
	Status                  string             `json:"status"`
	SourceDomainID          *string            `json:"source_domain_id,omitempty"`
	SourceTeamID            string             `json:"source_team_id"`
	TargetTeamID            string             `json:"target_team_id"`
	Region                  string             `json:"region"`
	CustomReturnPath        string             `json:"-"`
	TLS                     string             `json:"tls"`
	VerificationRecord      VerificationRecord `json:"record"`
	BlockedReason           *string            `json:"blocked_reason,omitempty"`
	FailureReason           *string            `json:"failure_reason,omitempty"`
	VerificationRequestedAt *time.Time         `json:"verification_requested_at,omitempty"`
	VerifiedAt              *time.Time         `json:"verified_at,omitempty"`
	CompletedAt             *time.Time         `json:"completed_at,omitempty"`
	ExpiresAt               time.Time          `json:"expires_at"`
	CreatedAt               time.Time          `json:"created_at"`
	UpdatedAt               time.Time          `json:"updated_at"`
}

type ReconciliationClaim struct {
	Claim Claim
}

type configuration struct {
	TLS              string
	CustomReturnPath string
}

type sourceDomain struct {
	ID               string
	TeamID           string
	Name             string
	Provider         string
	ProviderAccount  string
	ProviderRegion   string
	CustomReturnPath string
	Status           string
	CreatedBy        *string
	VerifiedAt       *time.Time
	CreatedAt        time.Time
}
