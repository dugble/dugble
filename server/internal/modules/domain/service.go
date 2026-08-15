package domain

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dugble/dugble/server/internal/authz"
	"github.com/dugble/dugble/server/internal/modules/emailtenant"
	platformemail "github.com/dugble/dugble/server/internal/platform/awsses"
	"github.com/dugble/dugble/server/internal/platform/systemmail"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

const manualHealthFailureReason = "sender domain verification checks no longer pass"

type emailTenantProvisioner interface {
	RequestProvisioning(context.Context, uuid.UUID, string) (emailtenant.Tenant, error)
}

type statusNotifier interface {
	SendSenderDomainStatus(context.Context, systemmail.SendSenderDomainStatusInput) error
}

type Service struct {
	db              *pgxpool.Pool
	repository      *Repository
	provider        platformemail.DomainProvider
	dns             platformemail.DNSVerifier
	tenantProvision emailTenantProvisioner
	notifier        statusNotifier
}

func (s *Service) WithNotifier(notifier statusNotifier) *Service {
	s.notifier = notifier
	return s
}

func (s *Service) WithDatabase(db *pgxpool.Pool) *Service {
	if s != nil {
		s.db = db
	}
	return s
}

type ReconciliationResult struct {
	Status              string
	VerificationRecords []VerificationRecord
}

func NewService(
	repository *Repository,
	provider platformemail.DomainProvider,
	dns platformemail.DNSVerifier,
	provisioners ...emailTenantProvisioner,
) *Service {
	var provisioner emailTenantProvisioner
	if len(provisioners) > 0 {
		provisioner = provisioners[0]
	}
	return &Service{
		repository:      repository,
		provider:        provider,
		dns:             dns,
		tenantProvision: provisioner,
	}
}

func (s *Service) List(ctx context.Context) ([]SenderDomain, error) {
	tc, err := requireTenantPermission(ctx, authz.PermissionSenderDomainsRead)
	if err != nil {
		return nil, err
	}
	domains, err := s.repository.List(ctx, tc.Scope.TeamID)
	if err != nil {
		return nil, apperrors.NewInternal("Unable to list sender domains", err)
	}
	return domains, nil
}

func (s *Service) Get(ctx context.Context, domainID string) (SenderDomain, error) {
	tc, err := requireTenantPermission(ctx, authz.PermissionSenderDomainsRead)
	if err != nil {
		return SenderDomain{}, err
	}
	id, err := parseDomainID(domainID)
	if err != nil {
		return SenderDomain{}, err
	}
	domain, err := s.repository.Get(ctx, id, tc.Scope.TeamID)
	if err != nil {
		return SenderDomain{}, apperrors.NewNotFound("Sender domain not found")
	}
	return domain, nil
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (CreateResult, error) {
	tc, err := requireTenantPermission(ctx, authz.PermissionSenderDomainsCreate)
	if err != nil {
		return CreateResult{}, err
	}
	domainName, region, configuration, err := validateCreate(req)
	if err != nil {
		return CreateResult{}, err
	}
	if s.provider == nil {
		return CreateResult{}, apperrors.NewInternal("Sender domain provider is not configured", nil)
	}
	if s.tenantProvision == nil {
		return CreateResult{}, apperrors.NewInternal("Customer email tenant provisioning is not configured", nil)
	}

	emailTenant, err := s.tenantProvision.RequestProvisioning(ctx, tc.Scope.TeamID, region)
	if err != nil {
		return CreateResult{}, apperrors.NewInternal("Unable to prepare customer email tenant", err)
	}
	if emailTenant.Status != emailtenant.StatusActive {
		return CreateResult{Provisioning: true}, nil
	}

	domain, err := s.createDomain(ctx, CreateDomainInput{
		TeamID:              tc.Scope.TeamID,
		Name:                domainName,
		Provider:            DefaultProvider,
		ProviderAccount:     DefaultProviderAccount,
		ProviderRegion:      region,
		CustomReturnPath:    configuration.CustomReturnPath,
		CreatedBy:           tc.Actor.UserIDPtr(),
		Configuration:       configuration,
		VerificationRecords: []VerificationRecord{},
	})
	if err != nil {
		if errors.Is(err, ErrSenderDomainAlreadyExists) {
			return CreateResult{}, apperrors.NewConflict("Sender domain already exists")
		}
		return CreateResult{}, apperrors.NewInternal("Unable to create sender domain", err)
	}
	id := uuid.MustParse(domain.ID)

	records, provisionErr := s.provider.ProvisionDomain(ctx, platformemail.DomainProvisionRequest{
		Domain:           domainName,
		Region:           region,
		CustomReturnPath: configuration.CustomReturnPath,
		SESTenantName:    emailTenant.ExternalName,
	})
	if provisionErr != nil {
		reason := provisionErr.Error()
		_, _ = s.updateVerification(
			ctx,
			id,
			tc.Scope.TeamID,
			StatusFailed,
			[]VerificationRecord{},
			&reason,
		)
		return CreateResult{}, apperrors.NewInternal("Unable to provision sender domain", provisionErr)
	}
	updated, saveErr := s.updateVerification(
		ctx,
		id,
		tc.Scope.TeamID,
		StatusPending,
		records,
		nil,
	)
	if saveErr == nil {
		return CreateResult{Domain: &updated}, nil
	}

	cleanupErr := s.provider.DeleteDomain(ctx, domainName, region)
	reason := "unable to save sender domain verification records"
	if cleanupErr != nil {
		reason = fmt.Sprintf("%s; provider cleanup failed: %v", reason, cleanupErr)
	}
	_, _ = s.updateVerification(
		ctx,
		id,
		tc.Scope.TeamID,
		StatusFailed,
		[]VerificationRecord{},
		&reason,
	)
	if cleanupErr != nil {
		saveErr = errors.Join(saveErr, cleanupErr)
	}
	return CreateResult{}, apperrors.NewInternal("Unable to save sender domain verification records", saveErr)
}

func (s *Service) Update(ctx context.Context, domainID string, req UpdateRequest) (SenderDomain, error) {
	tc, err := requireTenantPermission(ctx, authz.PermissionSenderDomainsCreate)
	if err != nil {
		return SenderDomain{}, err
	}
	id, err := parseDomainID(domainID)
	if err != nil {
		return SenderDomain{}, err
	}
	current, err := s.repository.Get(ctx, id, tc.Scope.TeamID)
	if err != nil {
		return SenderDomain{}, apperrors.NewNotFound("Sender domain not found")
	}
	if current.Status == StatusDisabled {
		return SenderDomain{}, apperrors.NewConflict("Disabled sender domains cannot be updated")
	}
	configuration, err := validateUpdate(current, req)
	if err != nil {
		return SenderDomain{}, err
	}
	updated, err := s.repository.UpdateConfiguration(ctx, id, tc.Scope.TeamID, configuration)
	if err != nil {
		return SenderDomain{}, apperrors.NewInternal("Unable to update sender domain", err)
	}
	return updated, nil
}

func (s *Service) Verify(ctx context.Context, domainID string) (SenderDomain, error) {
	tc, err := requireTenantPermission(ctx, authz.PermissionSenderDomainsCreate)
	if err != nil {
		return SenderDomain{}, err
	}
	id, err := parseDomainID(domainID)
	if err != nil {
		return SenderDomain{}, err
	}
	domain, err := s.repository.Get(ctx, id, tc.Scope.TeamID)
	if err != nil {
		return SenderDomain{}, apperrors.NewNotFound("Sender domain not found")
	}
	if domain.Status == StatusDisabled {
		return SenderDomain{}, apperrors.NewConflict("Disabled sender domains cannot be verified")
	}
	if s.provider == nil || s.dns == nil {
		return SenderDomain{}, apperrors.NewInternal("Sender domain verification is not configured", nil)
	}

	result, checkErr := s.Check(ctx, domain)
	if domain.Status == StatusVerified {
		records, reason := manualHealthObservation(domain, result, checkErr)
		updated, updateErr := s.updateManualHealthCheck(
			ctx,
			id,
			tc.Scope.TeamID,
			records,
			reason,
		)
		if updateErr != nil {
			return SenderDomain{}, apperrors.NewInternal("Unable to update sender domain health", updateErr)
		}
		s.notifyStatus(ctx, domain, updated)
		return updated, nil
	}
	if checkErr != nil {
		reason := checkErr.Error()
		updated, updateErr := s.updateVerification(
			ctx,
			id,
			tc.Scope.TeamID,
			StatusFailed,
			domain.VerificationRecords,
			&reason,
		)
		if updateErr != nil {
			return SenderDomain{}, apperrors.NewInternal("Unable to update sender domain verification", updateErr)
		}
		s.notifyStatus(ctx, domain, updated)
		return updated, nil
	}

	updated, err := s.updateVerification(
		ctx,
		id,
		tc.Scope.TeamID,
		result.Status,
		result.VerificationRecords,
		nil,
	)
	if err != nil {
		return SenderDomain{}, apperrors.NewInternal("Unable to update sender domain verification", err)
	}
	s.notifyStatus(ctx, domain, updated)
	return updated, nil
}

func (s *Service) notifyStatus(ctx context.Context, previous, updated SenderDomain) {
	status := NotificationStatus(previous, updated)
	if status == "" || s.notifier == nil {
		return
	}
	teamID, err := uuid.Parse(updated.TeamID)
	if err != nil {
		return
	}
	recipients, err := s.repository.ListNotificationRecipients(ctx, teamID)
	if err != nil {
		return
	}
	reason := ""
	if updated.FailureReason != nil {
		reason = *updated.FailureReason
	}
	for _, recipient := range recipients {
		_ = s.notifier.SendSenderDomainStatus(ctx, systemmail.SendSenderDomainStatusInput{ToEmail: recipient.Email, Name: recipient.Name, Domain: updated.Domain, Status: status, Reason: reason})
	}
}

func manualHealthObservation(
	domain SenderDomain,
	result ReconciliationResult,
	checkErr error,
) ([]VerificationRecord, *string) {
	// Verification records describe the successful verification event. Once the
	// domain is verified, health observations must not rewrite those records back
	// to pending because a recursive DNS resolver had a transient miss.
	records := verifiedVerificationRecords(domain.VerificationRecords)
	if checkErr != nil {
		reason := checkErr.Error()
		return records, &reason
	}
	if result.Status != StatusVerified {
		reason := manualHealthFailureReason
		return records, &reason
	}
	return records, nil
}

func verifiedVerificationRecords(records []VerificationRecord) []VerificationRecord {
	result := append([]VerificationRecord(nil), records...)
	for index := range result {
		result[index].Status = platformemail.RecordStatusVerified
	}
	return result
}

func (s *Service) Check(ctx context.Context, domain SenderDomain) (ReconciliationResult, error) {
	if s.provider == nil || s.dns == nil {
		return ReconciliationResult{}, errors.New("sender domain verification is not configured")
	}
	providerStatus, err := s.provider.GetDomainStatus(ctx, domain.Domain, domain.ProviderRegion)
	if err != nil {
		return ReconciliationResult{}, err
	}
	records := append([]VerificationRecord(nil), domain.VerificationRecords...)
	for index := range records {
		verified := false
		switch {
		case records[index].Record == platformemail.RecordDKIM:
			// SES DKIM SUCCESS already means the provider observed the BYODKIM
			// DNS record. Do not let a different recursive resolver override it.
			verified = providerStatus.DKIMVerified
		case records[index].Record == platformemail.RecordSPF && records[index].Type == platformemail.RecordTypeMX:
			// SES MAIL FROM SUCCESS means SES observed the required MX record.
			// The SPF TXT record remains independently checked below.
			verified = providerStatus.MailFromVerified
		default:
			verified = s.dns.Verify(ctx, domain.Domain, records[index])
		}
		records[index].Status = platformemail.RecordStatusPending
		if verified {
			records[index].Status = platformemail.RecordStatusVerified
		}
	}
	if providerStatus.IdentityVerified && !providerStatus.MailFromConfigured {
		configurator, ok := s.provider.(platformemail.DomainMailFromConfigurator)
		if !ok {
			return ReconciliationResult{}, errors.New("sender domain provider does not support MAIL FROM configuration")
		}
		if configureErr := configurator.ConfigureDomainMailFrom(
			ctx,
			domain.Domain,
			domain.ProviderRegion,
			domain.CustomReturnPath,
		); configureErr != nil {
			return ReconciliationResult{}, configureErr
		}
		return ReconciliationResult{
			Status:              StatusPending,
			VerificationRecords: records,
		}, nil
	}
	status := verificationStatus(records, providerStatus)
	if status == StatusVerified {
		associator, ok := s.provider.(platformemail.DomainTenantAssociator)
		if !ok {
			return ReconciliationResult{}, errors.New("sender domain provider does not support tenant association")
		}
		teamID, parseErr := uuid.Parse(domain.TeamID)
		if parseErr != nil {
			return ReconciliationResult{}, fmt.Errorf("parse sender domain team id: %w", parseErr)
		}
		if associationErr := associator.AssociateDomainWithTenant(
			ctx,
			domain.Domain,
			domain.ProviderRegion,
			emailtenant.AWSExternalName(teamID),
		); associationErr != nil {
			return ReconciliationResult{}, associationErr
		}
	}
	return ReconciliationResult{
		Status:              status,
		VerificationRecords: records,
	}, nil
}

func verificationStatus(records []VerificationRecord, providerStatus platformemail.DomainStatus) string {
	if len(records) == 0 {
		return StatusPending
	}
	for _, record := range records {
		if record.Status != platformemail.RecordStatusVerified {
			return StatusPending
		}
	}
	if providerStatus.IdentityVerified && providerStatus.DKIMVerified && providerStatus.MailFromVerified {
		return StatusVerified
	}
	return StatusPending
}

func (s *Service) Delete(ctx context.Context, domainID string) (SenderDomain, error) {
	tc, err := requireTenantPermission(ctx, authz.PermissionSenderDomainsDelete)
	if err != nil {
		return SenderDomain{}, err
	}
	id, err := parseDomainID(domainID)
	if err != nil {
		return SenderDomain{}, err
	}
	if s.provider == nil {
		return SenderDomain{}, apperrors.NewInternal("Sender domain provider is not configured", nil)
	}

	domain, err := s.repository.Disable(ctx, id, tc.Scope.TeamID)
	if err != nil {
		return SenderDomain{}, apperrors.NewNotFound("Sender domain not found")
	}
	if disassociator, ok := s.provider.(platformemail.DomainTenantDisassociator); ok {
		teamID, parseErr := uuid.Parse(domain.TeamID)
		if parseErr != nil {
			return domain, apperrors.NewInternal("Unable to resolve sender domain tenant for provider deletion", parseErr)
		}
		if disassociateErr := disassociator.DisassociateDomainFromTenant(
			ctx,
			domain.Domain,
			domain.ProviderRegion,
			emailtenant.AWSExternalName(teamID),
		); disassociateErr != nil {
			return domain, apperrors.NewInternal(
				"Unable to detach sender domain from provider tenant; the domain has been disabled",
				disassociateErr,
			)
		}
	}
	if err := s.provider.DeleteDomain(ctx, domain.Domain, domain.ProviderRegion); err != nil {
		return domain, apperrors.NewInternal(
			"Unable to delete sender domain from provider; the domain has been disabled",
			err,
		)
	}
	purged, err := s.repository.PurgeIfUnreferenced(ctx, id, tc.Scope.TeamID)
	if err != nil {
		return domain, apperrors.NewInternal(
			"Sender domain was removed from provider but local cleanup failed",
			err,
		)
	}
	if purged {
		return domain, nil
	}
	return s.repository.Get(ctx, id, tc.Scope.TeamID)
}

func (s *Service) withRepositoryTx(
	ctx context.Context,
	operation string,
	fn func(*Repository) (SenderDomain, error),
) (SenderDomain, error) {
	if s == nil || s.db == nil || s.repository == nil {
		return SenderDomain{}, errors.New("sender domain transaction service is not configured")
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return SenderDomain{}, fmt.Errorf("begin %s: %w", operation, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	value, err := fn(s.repository.WithTx(tx))
	if err != nil {
		return SenderDomain{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return SenderDomain{}, fmt.Errorf("commit %s: %w", operation, err)
	}
	return value, nil
}

func (s *Service) createDomain(ctx context.Context, input CreateDomainInput) (SenderDomain, error) {
	return s.withRepositoryTx(ctx, "sender domain creation", func(repository *Repository) (SenderDomain, error) {
		return repository.Create(ctx, input)
	})
}

func (s *Service) updateVerification(
	ctx context.Context,
	id, teamID uuid.UUID,
	status string,
	records []VerificationRecord,
	failureReason *string,
) (SenderDomain, error) {
	return s.withRepositoryTx(ctx, "sender domain verification update", func(repository *Repository) (SenderDomain, error) {
		return repository.UpdateVerification(ctx, id, teamID, status, records, failureReason)
	})
}

func (s *Service) updateManualHealthCheck(
	ctx context.Context,
	id, teamID uuid.UUID,
	records []VerificationRecord,
	failureReason *string,
) (SenderDomain, error) {
	return s.withRepositoryTx(ctx, "sender domain manual health update", func(repository *Repository) (SenderDomain, error) {
		return repository.UpdateManualHealthCheck(ctx, id, teamID, records, failureReason)
	})
}

func requireTenantPermission(ctx context.Context, permission authz.Permission) (authz.Access, error) {
	tc, decision := authz.ResolveAccess(ctx, permission)
	if !decision.Allowed {
		return authz.Access{}, apperrors.NewForbidden(decision.Reason)
	}
	return tc, nil
}
