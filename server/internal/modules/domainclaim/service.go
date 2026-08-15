package domainclaim

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dugble/dugble/server/internal/authz"
	"github.com/dugble/dugble/server/internal/modules/emailtenant"
	platformemail "github.com/dugble/dugble/server/internal/platform/awsses"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

type emailTenantProvisioner interface {
	RequestProvisioning(context.Context, uuid.UUID, string) (emailtenant.Tenant, error)
}

type Service struct {
	repository      *Repository
	provider        platformemail.DomainProvider
	dns             platformemail.DNSVerifier
	tenantProvision emailTenantProvisioner
	now             func() time.Time
}

func NewService(repository *Repository, provider platformemail.DomainProvider, dns platformemail.DNSVerifier, tenantProvision emailTenantProvisioner) *Service {
	return &Service{
		repository: repository, provider: provider, dns: dns, tenantProvision: tenantProvision,
		now: func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Start(ctx context.Context, req Request) (Claim, error) {
	access, err := requireTenantPermission(ctx, authz.PermissionSenderDomainsCreate)
	if err != nil {
		return Claim{}, err
	}
	name, region, cfg, err := validateRequest(req)
	if err != nil {
		return Claim{}, err
	}
	claim, err := s.repository.Create(ctx, access.Scope.TeamID, access.Actor.UserID, name, region, cfg)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Claim{}, apperrors.NewNotFound("Sender domain is not currently owned by another team")
	case errors.Is(err, ErrAlreadyOwned):
		return Claim{}, apperrors.NewConflict("Sender domain is already owned by this team")
	case errors.Is(err, ErrClaimAlreadyExists):
		return Claim{}, apperrors.NewConflict("An active claim already exists for this sender domain")
	case err != nil:
		return Claim{}, apperrors.NewInternal("Unable to create sender domain claim", err)
	default:
		return claim, nil
	}
}

func (s *Service) Get(ctx context.Context, domainID string) (Claim, error) {
	access, err := requireTenantPermission(ctx, authz.PermissionSenderDomainsRead)
	if err != nil {
		return Claim{}, err
	}
	id, err := parseDomainID(domainID)
	if err != nil {
		return Claim{}, err
	}
	claim, err := s.repository.Get(ctx, id, access.Scope.TeamID)
	if err != nil {
		return Claim{}, apperrors.NewNotFound("Sender domain claim not found")
	}
	return claim, nil
}

func (s *Service) Verify(ctx context.Context, domainID string) (Claim, error) {
	access, err := requireTenantPermission(ctx, authz.PermissionSenderDomainsCreate)
	if err != nil {
		return Claim{}, err
	}
	id, err := parseDomainID(domainID)
	if err != nil {
		return Claim{}, err
	}
	claim, err := s.repository.RequestVerification(ctx, id, access.Scope.TeamID)
	if err != nil {
		return Claim{}, apperrors.NewNotFound("Active sender domain claim not found")
	}
	return claim, nil
}

func (s *Service) Cancel(ctx context.Context, domainID string) (Claim, error) {
	access, err := requireTenantPermission(ctx, authz.PermissionSenderDomainsDelete)
	if err != nil {
		return Claim{}, err
	}
	id, err := parseDomainID(domainID)
	if err != nil {
		return Claim{}, err
	}
	claim, err := s.repository.Cancel(ctx, id, access.Scope.TeamID)
	if err != nil {
		return Claim{}, apperrors.NewNotFound("Active sender domain claim not found")
	}
	return claim, nil
}

// Reconcile performs one idempotent ownership-transfer iteration for a locked claim.
func (s *Service) Reconcile(ctx context.Context, claim Claim, workerID string) (Claim, error) {
	if s == nil || s.repository == nil || s.provider == nil || s.dns == nil || s.tenantProvision == nil {
		return Claim{}, errors.New("domain claim reconciliation is not configured")
	}
	claimID, err := uuid.Parse(claim.ID)
	if err != nil {
		return Claim{}, fmt.Errorf("parse domain claim id: %w", err)
	}
	if s.now().After(claim.ExpiresAt) {
		return s.repository.MarkFailed(ctx, claimID, workerID, errors.New("domain claim expired"))
	}
	if !s.dns.Verify(ctx, claim.Name, claim.VerificationRecord) {
		return s.repository.Release(ctx, claimID, workerID)
	}

	source, err := s.repository.Source(ctx, claim)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return s.repository.MarkFailed(ctx, claimID, workerID, errors.New("claimed source domain no longer exists"))
		}
		return Claim{}, fmt.Errorf("load claimed source domain: %w", err)
	}
	sourceID, err := uuid.Parse(source.ID)
	if err != nil {
		return Claim{}, fmt.Errorf("parse claimed source domain id: %w", err)
	}

	ownershipSince := source.CreatedAt
	if source.VerifiedAt != nil {
		ownershipSince = *source.VerifiedAt
	}
	if s.now().Before(ownershipSince.Add(DefaultOwnershipGracePeriod)) {
		return s.repository.MarkBlocked(ctx, claimID, workerID, BlockedGracePeriod)
	}
	recent, err := s.repository.HasRecentOwnerActivity(ctx, sourceID, s.now().Add(-DefaultRecentOwnerActivityWindow))
	if err != nil {
		return Claim{}, err
	}
	if recent {
		return s.repository.MarkBlocked(ctx, claimID, workerID, BlockedRecentOwnerActivity)
	}
	pending, err := s.repository.HasPendingScheduledEmails(ctx, sourceID)
	if err != nil {
		return Claim{}, err
	}
	if pending {
		return s.repository.MarkBlocked(ctx, claimID, workerID, BlockedPendingScheduledEmails)
	}

	verified, err := s.repository.MarkVerified(ctx, claimID, workerID)
	if err != nil {
		return Claim{}, fmt.Errorf("mark domain claim verified: %w", err)
	}
	targetTeamID, err := uuid.Parse(verified.TargetTeamID)
	if err != nil {
		return Claim{}, fmt.Errorf("parse claim target team id: %w", err)
	}
	emailTenant, err := s.tenantProvision.RequestProvisioning(ctx, targetTeamID, verified.Region)
	if err != nil {
		return Claim{}, fmt.Errorf("prepare claimed domain email tenant: %w", err)
	}
	if emailTenant.Status != emailtenant.StatusActive {
		return s.repository.Release(ctx, claimID, workerID)
	}

	// Freeze the source before touching SES so new sends cannot race the transfer.
	if source.Status != "disabled" {
		if err := s.repository.FreezeSource(ctx, source); err != nil {
			return Claim{}, err
		}
	}
	sourceTeamID, err := uuid.Parse(source.TeamID)
	if err != nil {
		return Claim{}, fmt.Errorf("parse source team id: %w", err)
	}
	if disassociator, ok := s.provider.(platformemail.DomainTenantDisassociator); ok {
		if err := disassociator.DisassociateDomainFromTenant(ctx, source.Name, source.ProviderRegion, emailtenant.AWSExternalName(sourceTeamID)); err != nil {
			return Claim{}, fmt.Errorf("detach claimed domain from source tenant: %w", err)
		}
	}
	if err := s.provider.DeleteDomain(ctx, source.Name, source.ProviderRegion); err != nil {
		return Claim{}, fmt.Errorf("remove claimed domain from source provider route: %w", err)
	}
	records, err := s.provider.ProvisionDomain(ctx, platformemail.DomainProvisionRequest{
		Domain: verified.Name, Region: verified.Region, CustomReturnPath: verified.CustomReturnPath, SESTenantName: emailTenant.ExternalName,
	})
	if err != nil {
		return Claim{}, fmt.Errorf("provision claimed domain for target team: %w", err)
	}
	completed, err := s.repository.CompleteTransfer(ctx, verified, workerID, records)
	if err != nil {
		return Claim{}, err
	}
	return completed, nil
}

func requireTenantPermission(ctx context.Context, permission authz.Permission) (authz.Access, error) {
	access, decision := authz.ResolveAccess(ctx, permission)
	if !decision.Allowed {
		return authz.Access{}, apperrors.NewForbidden(decision.Reason)
	}
	return access, nil
}
