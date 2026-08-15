package emailtenant

import (
	"context"
	"errors"
	"fmt"
	"strings"

	platformemail "github.com/dugble/dugble/server/internal/platform/awsses"
	"github.com/google/uuid"
)

type ProvisioningRequest struct {
	EventID          uuid.UUID
	TenantID         uuid.UUID
	TeamID           uuid.UUID
	Provider         string
	Region           string
	ExternalName     string
	SuppressionScope string
	ReputationPolicy string
}

type tenantStore interface {
	BeginTx(context.Context) (Transaction, error)
	CreateTx(context.Context, Transaction, CreateParams) (Tenant, error)
	MarkProvisioningTx(context.Context, Transaction, uuid.UUID) (Tenant, error)
}

type provisioningQueue interface {
	EnqueueProvisioningTx(context.Context, Transaction, ProvisioningRequest) error
}

type Service struct {
	repository tenantStore
	queue      provisioningQueue
}

func NewService(repository tenantStore, queue provisioningQueue) *Service {
	return &Service{repository: repository, queue: queue}
}

// RequestProvisioning reserves one regional provider tenant for a team and
// atomically publishes a provisioning command through the PostgreSQL outbox.
func (service *Service) RequestProvisioning(ctx context.Context, teamID uuid.UUID, region string) (Tenant, error) {
	if service == nil || service.repository == nil {
		return Tenant{}, errors.New("email tenant service is not configured")
	}
	if service.queue == nil {
		return Tenant{}, errors.New("email tenant provisioning queue is not configured")
	}
	if teamID == uuid.Nil {
		return Tenant{}, errors.New("email tenant team id is required")
	}
	region, supported := platformemail.NormalizeSESRegion(region)
	if !supported {
		return Tenant{}, fmt.Errorf("unsupported SES region %q", region)
	}

	tx, err := service.repository.BeginTx(ctx)
	if err != nil {
		return Tenant{}, fmt.Errorf("begin email tenant provisioning transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tenant, err := service.repository.CreateTx(ctx, tx, CreateParams{
		TeamID:           teamID,
		Provider:         ProviderAWSSES,
		Region:           region,
		ExternalName:     AWSExternalName(teamID),
		SuppressionScope: SuppressionScopeTenant,
		ReputationPolicy: ReputationPolicyStandard,
	})
	if err != nil {
		return Tenant{}, err
	}

	switch tenant.Status {
	case StatusProvisioning, StatusActive, StatusPaused, StatusDeleting:
		if err := tx.Commit(ctx); err != nil {
			return Tenant{}, fmt.Errorf("commit existing email tenant transaction: %w", err)
		}
		return tenant, nil
	case StatusPending, StatusFailed:
	default:
		return Tenant{}, fmt.Errorf("unsupported email tenant status %q", tenant.Status)
	}

	tenant, err = service.repository.MarkProvisioningTx(ctx, tx, tenant.ID)
	if err != nil {
		return Tenant{}, err
	}
	request := ProvisioningRequest{
		EventID:          uuid.New(),
		TenantID:         tenant.ID,
		TeamID:           tenant.TeamID,
		Provider:         tenant.Provider,
		Region:           tenant.Region,
		ExternalName:     tenant.ExternalName,
		SuppressionScope: tenant.SuppressionScope,
		ReputationPolicy: tenant.ReputationPolicy,
	}
	if err := service.queue.EnqueueProvisioningTx(ctx, tx, request); err != nil {
		return Tenant{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Tenant{}, fmt.Errorf("commit email tenant provisioning transaction: %w", err)
	}
	return tenant, nil
}

const awsTenantNamePrefix = "dugble-t-"

// AWSExternalName derives a stable, opaque SES tenant name from the immutable
// Dugble team ID. Team names and domains are deliberately excluded because they
// can change during the lifetime of the provider tenant.
func AWSExternalName(teamID uuid.UUID) string {
	return awsTenantNamePrefix + strings.ReplaceAll(teamID.String(), "-", "")
}

func ParseTeamID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse email tenant team id: %w", err)
	}
	return id, nil
}
