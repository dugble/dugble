package emailtenant

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	ProviderAWSSES = "aws_ses"

	StatusPending      = "pending"
	StatusProvisioning = "provisioning"
	StatusActive       = "active"
	StatusPaused       = "paused"
	StatusDeleting     = "deleting"
	StatusFailed       = "failed"

	SuppressionScopeAccount = "account"
	SuppressionScopeTenant  = "tenant"

	ReputationPolicyNone     = "none"
	ReputationPolicyStandard = "standard"
	ReputationPolicyStrict   = "strict"

	ProvisionSubject    = "dugble.job.email.tenant.provision.v1"
	ProvisionEventType  = "email.tenant.provision.requested.v1"
	ProvisionConsumer   = "dugble-email-tenant-provision-v1"
	ProvisionDLQSubject = "dugble.dlq.email.tenant.provision.v1"
)

type Tenant struct {
	ID               uuid.UUID `json:"id"`
	TeamID           uuid.UUID `json:"team_id"`
	Provider         string    `json:"provider"`
	Region           string    `json:"region"`
	ExternalName     string    `json:"external_name"`
	ExternalID       *string   `json:"external_id,omitempty"`
	TenantARN        *string   `json:"tenant_arn,omitempty"`
	Status           string    `json:"status"`
	SuppressionScope string    `json:"suppression_scope"`
	ReputationPolicy string    `json:"reputation_policy"`
	FailureReason    *string   `json:"failure_reason,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type CreateParams struct {
	TeamID           uuid.UUID
	Provider         string
	Region           string
	ExternalName     string
	SuppressionScope string
	ReputationPolicy string
}

type ProvisionRequest struct {
	Region           string
	ExternalName     string
	SuppressionScope string
	ReputationPolicy string
}

type ProvisionResult struct {
	ExternalID string
	TenantARN  string
}

type Provisioner interface {
	ProvisionTenant(context.Context, ProvisionRequest) (ProvisionResult, error)
}

type ProvisioningCommand struct {
	EventID          uuid.UUID `json:"event_id"`
	TenantID         uuid.UUID `json:"tenant_id"`
	TeamID           uuid.UUID `json:"team_id"`
	Provider         string    `json:"provider"`
	Region           string    `json:"region"`
	ExternalName     string    `json:"external_name"`
	SuppressionScope string    `json:"suppression_scope"`
	ReputationPolicy string    `json:"reputation_policy"`
	SchemaVersion    int       `json:"schema_version"`
}
