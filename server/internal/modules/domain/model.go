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

	DefaultHealthFailureThreshold int32 = 3
)

type VerificationRecord = platformemail.VerificationRecord

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
	TLS                       string               `json:"tls"`
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

func NotificationStatus(previous, updated SenderDomain) string {
	if updated.Status == StatusVerified && previous.Status != StatusVerified {
		return StatusVerified
	}
	if updated.Status == StatusFailed && previous.Status != StatusFailed {
		return StatusFailed
	}
	if updated.HealthStatus == HealthStatusDegraded && previous.HealthStatus != HealthStatusDegraded {
		return HealthStatusDegraded
	}
	return ""
}

type CreateRequest struct {
	Name   string `json:"name"`
	Region string `json:"region"`
	TLS    string `json:"tls,omitempty"`
}

type UpdateRequest struct {
	TLS *string `json:"tls,omitempty"`
}

type DomainConfiguration struct {
	TLS              string
	CustomReturnPath string
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
