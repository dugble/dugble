package domain

import (
	"time"

	platformemail "github.com/dugble/dugble/server/internal/platform/awsses"
)

const (
	DefaultProvider         = "aws_ses"
	DefaultProviderAccount  = "default"
	DefaultCustomReturnPath = "send"
	DefaultTLSMode          = "opportunistic"

	StatusNotStarted        = "not_started"
	StatusPending           = "pending"
	StatusVerified          = "verified"
	StatusPartiallyVerified = "partially_verified"
	StatusPartiallyFailed   = "partially_failed"
	StatusFailed            = "failed"
	StatusTemporaryFailure  = "temporary_failure"
	StatusDisabled          = "disabled"

	HealthStatusUnknown  = "unknown"
	HealthStatusHealthy  = "healthy"
	HealthStatusDegraded = "degraded"

	ClaimStatusPending    = "pending"
	ClaimStatusVerified   = "verified"
	ClaimStatusCompleted  = "completed"
	ClaimStatusBlocked    = "blocked"
	ClaimStatusExpired    = "expired"
	ClaimStatusSuperseded = "superseded"
	ClaimStatusCanceled   = "canceled"
	ClaimStatusFailed     = "failed"

	ClaimBlockedGracePeriod            = "grace_period"
	ClaimBlockedRecentOwnerActivity    = "recent_owner_activity"
	ClaimBlockedPendingScheduledEmails = "pending_scheduled_emails"

	DefaultHealthFailureThreshold int32 = 3
	DefaultClaimLifetime                = 7 * 24 * time.Hour
)

type VerificationRecord = platformemail.VerificationRecord

// Capabilities is retained internally for compatibility with the existing
// persistence model. Sender domains are outbound-only, so callers cannot
// configure capabilities through the public API.
type Capabilities struct {
	Sending   bool `json:"-"`
	Receiving bool `json:"-"`
}

type SenderDomain struct {
	ID                        string               `json:"id"`
	TeamID                    string               `json:"team_id"`
	Domain                    string               `json:"name"`
	Provider                  string               `json:"provider,omitempty"`
	ProviderAccount           string               `json:"provider_account,omitempty"`
	ProviderRegion            string               `json:"region"`
	ProviderExternalID        *string              `json:"provider_external_id,omitempty"`
	Status                    string               `json:"status"`
	ProviderStatus            *string              `json:"provider_status,omitempty"`
	VerificationRecords       []VerificationRecord `json:"records"`
	OpenTracking              bool                 `json:"-"`
	ClickTracking             bool                 `json:"-"`
	TrackingSubdomain         *string              `json:"-"`
	ActiveTrackingSubdomain   *string              `json:"-"`
	TLS                       string               `json:"tls"`
	Capabilities              Capabilities         `json:"-"`
	CustomReturnPath          string               `json:"-"`
	FailureReason             *string              `json:"failure_reason,omitempty"`
	HealthStatus              string               `json:"health_status"`
	ConsecutiveHealthFailures int32                `json:"consecutive_health_failures"`
	LastCheckedAt             *time.Time           `json:"last_checked_at,omitempty"`
	LastHealthCheckedAt       *time.Time           `json:"last_health_checked_at,omitempty"`
	LastHealthFailureAt       *time.Time           `json:"last_health_failure_at,omitempty"`
	VerifiedAt                *time.Time           `json:"verified_at,omitempty"`
	DisabledAt                *time.Time           `json:"disabled_at,omitempty"`
	CreatedBy                 *string              `json:"created_by,omitempty"`
	CreatedAt                 time.Time            `json:"created_at"`
	UpdatedAt                 time.Time            `json:"updated_at"`
}

// CreateRequest intentionally exposes only the sender-domain settings that are
// part of the supported product contract. MAIL FROM, tracking, and capabilities
// are platform-owned defaults.
type CreateRequest struct {
	Name   string `json:"name"`
	Region string `json:"region"`
	TLS    string `json:"tls,omitempty"`
}

type UpdateRequest struct {
	TLS *string `json:"tls,omitempty"`
}

// DomainConfiguration keeps legacy persistence fields internal while the
// public API remains intentionally small. These defaults are not customer
// configurable.
type DomainConfiguration struct {
	OpenTracking      bool
	ClickTracking     bool
	TrackingSubdomain *string
	TLS               string
	Capabilities      Capabilities
	CustomReturnPath  string
}

type CreateResult struct {
	Domain       *SenderDomain
	Provisioning bool
}

type ProvisioningResponse struct {
	Status            string `json:"status"`
	Message           string `json:"message"`
	RetryAfterSeconds int    `json:"retry_after_seconds"`
}

type ClaimRequest struct {
	Name   string `json:"name"`
	Region string `json:"region"`
	TLS    string `json:"tls,omitempty"`
}

type DomainClaim struct {
	ID                      string             `json:"id"`
	DomainID                string             `json:"domain_id"`
	Name                    string             `json:"name"`
	Status                  string             `json:"status"`
	SourceDomainID          *string            `json:"source_domain_id,omitempty"`
	SourceTeamID            string             `json:"source_team_id"`
	TargetTeamID            string             `json:"target_team_id"`
	Region                  string             `json:"region"`
	CustomReturnPath        string             `json:"-"`
	OpenTracking            bool               `json:"-"`
	ClickTracking           bool               `json:"-"`
	TrackingSubdomain       *string            `json:"-"`
	TLS                     string             `json:"tls"`
	Capabilities            Capabilities       `json:"-"`
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
