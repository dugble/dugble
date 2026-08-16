package emailtenant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dugble/dugble/server/internal/platform/outbox"
)

var (
	ErrProvisioningQueueNotConfigured     = errors.New("email tenant provisioning queue is not configured")
	ErrProvisioningProcessorNotConfigured = errors.New("email tenant provisioning processor is not configured")
	ErrProvisioningTransactionRequired    = errors.New("email tenant provisioning requires a PostgreSQL transaction")
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
	CreateTx(context.Context, Transaction, CreateParams) (Tenant, error)
	MarkProvisioningTx(context.Context, Transaction, uuid.UUID) (Tenant, error)
}

type provisioningQueue interface {
	EnqueueProvisioningTx(context.Context, Transaction, ProvisioningRequest) error
}

type Service struct {
	db         *pgxpool.Pool
	repository tenantStore
	queue      provisioningQueue
}

func NewService(db *pgxpool.Pool, repository tenantStore, queue provisioningQueue) *Service {
	return &Service{db: db, repository: repository, queue: queue}
}

func (service *Service) RequestProvisioning(ctx context.Context, teamID uuid.UUID, region string) (Tenant, error) {
	if service == nil || service.db == nil || service.repository == nil {
		return Tenant{}, errors.New("email tenant service is not configured")
	}
	if service.queue == nil {
		return Tenant{}, errors.New("email tenant provisioning queue is not configured")
	}
	if teamID == uuid.Nil {
		return Tenant{}, errors.New("email tenant team id is required")
	}
	region = strings.ToLower(strings.TrimSpace(region))
	if !isSupportedSESRegion(region) {
		return Tenant{}, fmt.Errorf("unsupported SES region %q", region)
	}

	tx, err := service.db.BeginTx(ctx, pgx.TxOptions{})
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

func isSupportedSESRegion(region string) bool {
	switch region {
	case "us-east-1", "eu-north-1":
		return true
	default:
		return false
	}
}

const awsTenantNamePrefix = "dugble-t-"

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

type provisioningOutboxStore interface {
	EnqueueTx(context.Context, pgx.Tx, outbox.Event) (uuid.UUID, error)
}

type ProvisioningQueue struct {
	store provisioningOutboxStore
}

func NewProvisioningQueue(store provisioningOutboxStore) *ProvisioningQueue {
	return &ProvisioningQueue{store: store}
}

func (queue *ProvisioningQueue) EnqueueProvisioningTx(ctx context.Context, tx Transaction, request ProvisioningRequest) error {
	if queue == nil || queue.store == nil {
		return ErrProvisioningQueueNotConfigured
	}
	pgxTx, ok := tx.(pgx.Tx)
	if !ok || pgxTx == nil {
		return ErrProvisioningTransactionRequired
	}
	command := provisioningCommandFromRequest(request)
	if err := ValidateProvisioningCommand(command); err != nil {
		return err
	}
	payload, err := json.Marshal(command)
	if err != nil {
		return fmt.Errorf("encode email tenant provisioning command: %w", err)
	}
	_, err = queue.store.EnqueueTx(ctx, pgxTx, outbox.Event{
		ID:            command.EventID,
		Subject:       ProvisionSubject,
		AggregateType: "email_tenant",
		AggregateID:   command.TenantID,
		Payload:       payload,
		Headers: map[string]string{
			"Dugble-Event-Type": ProvisionEventType,
		},
	})
	if err != nil {
		return fmt.Errorf("enqueue email tenant provisioning command: %w", err)
	}
	return nil
}

func provisioningCommandFromRequest(request ProvisioningRequest) ProvisioningCommand {
	return ProvisioningCommand{
		EventID:          request.EventID,
		TenantID:         request.TenantID,
		TeamID:           request.TeamID,
		Provider:         request.Provider,
		Region:           request.Region,
		ExternalName:     request.ExternalName,
		SuppressionScope: request.SuppressionScope,
		ReputationPolicy: request.ReputationPolicy,
		SchemaVersion:    1,
	}
}

type provisioningStore interface {
	Get(context.Context, uuid.UUID) (Tenant, error)
	MarkActive(context.Context, uuid.UUID, string, string) (Tenant, error)
	MarkFailed(context.Context, uuid.UUID, error) (Tenant, error)
}

type ProvisioningProcessor struct {
	store    provisioningStore
	provider Provisioner
}

func NewProvisioningProcessor(store provisioningStore, provider Provisioner) *ProvisioningProcessor {
	return &ProvisioningProcessor{store: store, provider: provider}
}

func (processor *ProvisioningProcessor) Handle(ctx context.Context, command ProvisioningCommand) error {
	if processor == nil || processor.store == nil || processor.provider == nil {
		return ErrProvisioningProcessorNotConfigured
	}
	if err := ValidateProvisioningCommand(command); err != nil {
		return err
	}
	current, err := processor.store.Get(ctx, command.TenantID)
	if err != nil {
		return fmt.Errorf("load email tenant for provisioning: %w", err)
	}
	if current.TeamID != command.TeamID || current.Provider != command.Provider || current.Region != command.Region || current.ExternalName != command.ExternalName {
		return errors.New("email tenant provisioning command does not match persisted tenant")
	}
	if current.Status == StatusActive {
		return nil
	}
	if current.Status != StatusProvisioning {
		return fmt.Errorf("email tenant is %s, expected provisioning", current.Status)
	}

	result, err := processor.provider.ProvisionTenant(ctx, ProvisionRequest{
		Region:           current.Region,
		ExternalName:     current.ExternalName,
		SuppressionScope: current.SuppressionScope,
		ReputationPolicy: current.ReputationPolicy,
	})
	if err != nil {
		return fmt.Errorf("provision SES tenant: %w", err)
	}
	if _, err := processor.store.MarkActive(ctx, current.ID, result.ExternalID, result.TenantARN); err != nil {
		return fmt.Errorf("activate email tenant: %w", err)
	}
	return nil
}

func (processor *ProvisioningProcessor) HandleExhausted(ctx context.Context, command ProvisioningCommand, cause error) error {
	if processor == nil || processor.store == nil {
		return ErrProvisioningProcessorNotConfigured
	}
	current, err := processor.store.Get(ctx, command.TenantID)
	if err != nil {
		return fmt.Errorf("load exhausted email tenant: %w", err)
	}
	if current.Status == StatusActive || current.Status == StatusFailed {
		return nil
	}
	if _, err := processor.store.MarkFailed(ctx, command.TenantID, cause); err != nil {
		return fmt.Errorf("mark email tenant provisioning failed: %w", err)
	}
	return nil
}
