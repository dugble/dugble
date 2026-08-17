package senderid

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/dugble/dugble/server/internal/authz"
	"github.com/dugble/dugble/server/internal/platform/systemmail"
	relaysenderid "github.com/dugble/dugble/server/internal/relay/senderid"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

const (
	maxPurposeLength  = 500
	maxProviderLength = 120
)

type Service struct {
	repository *Repository
	providers  []string
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

// WithProviders configures provider preference for newly-created Sender IDs.
// The first provider is used when the API request omits an explicit provider.
func (s *Service) WithProviders(providers ...string) *Service {
	if s == nil {
		return nil
	}
	seen := make(map[string]struct{}, len(providers))
	s.providers = s.providers[:0]
	for _, provider := range providers {
		provider = strings.ToLower(strings.TrimSpace(provider))
		if provider == "" {
			continue
		}
		if _, exists := seen[provider]; exists {
			continue
		}
		seen[provider] = struct{}{}
		s.providers = append(s.providers, provider)
	}
	return s
}

func (s *Service) List(ctx context.Context) ([]SenderID, error) {
	tenantContext, err := requireTenantPermission(ctx, authz.PermissionSenderIDsRead)
	if err != nil {
		return nil, err
	}
	senderIDs, err := s.repository.List(ctx, tenantContext.Scope.TeamID)
	if err != nil {
		return nil, apperrors.NewInternal("Unable to list sender IDs", err)
	}
	return senderIDs, nil
}

func (s *Service) Get(ctx context.Context, senderID string) (SenderID, error) {
	tenantContext, err := requireTenantPermission(ctx, authz.PermissionSenderIDsRead)
	if err != nil {
		return SenderID{}, err
	}
	parsedSenderID, err := uuid.Parse(strings.TrimSpace(senderID))
	if err != nil {
		return SenderID{}, apperrors.NewBadRequest("Sender ID id must be a valid UUID")
	}
	value, err := s.repository.Get(ctx, parsedSenderID, tenantContext.Scope.TeamID)
	if err != nil {
		return SenderID{}, apperrors.NewNotFound("Sender ID not found")
	}
	return value, nil
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (SenderID, error) {
	tenantContext, err := requireTenantPermission(ctx, authz.PermissionSenderIDsCreate)
	if err != nil {
		return SenderID{}, err
	}
	name, countryCode, purpose, provider, err := s.validateCreate(req)
	if err != nil {
		return SenderID{}, err
	}
	senderID, err := s.repository.Create(
		ctx,
		tenantContext.Scope.TeamID,
		name,
		countryCode,
		purpose,
		provider,
		tenantContext.Actor.UserIDPtr(),
	)
	if err != nil {
		if errors.Is(err, ErrSenderIDAlreadyExists) {
			return SenderID{}, apperrors.NewConflict("Sender ID already exists for this team and country")
		}
		return SenderID{}, apperrors.NewInternal("Unable to create sender ID", err)
	}
	return senderID, nil
}

func (s *Service) Delete(ctx context.Context, senderID string) (SenderID, error) {
	tenantContext, err := requireTenantPermission(ctx, authz.PermissionSenderIDsDelete)
	if err != nil {
		return SenderID{}, err
	}
	parsedSenderID, err := uuid.Parse(strings.TrimSpace(senderID))
	if err != nil {
		return SenderID{}, apperrors.NewBadRequest("Sender ID id must be a valid UUID")
	}
	value, err := s.repository.Deactivate(ctx, parsedSenderID, tenantContext.Scope.TeamID)
	if err != nil {
		return SenderID{}, apperrors.NewNotFound("Sender ID not found")
	}
	return value, nil
}

type reconciliationNotifier interface {
	SendSenderIDStatus(context.Context, systemmail.SendSenderIDStatusInput) error
}

type ReconciliationService struct {
	repository           *Repository
	providerTimeout      time.Duration
	pendingCheckInterval time.Duration
	retryBaseInterval    time.Duration
	maxRetryInterval     time.Duration
	workerID             string
	now                  func() time.Time
	notifier             reconciliationNotifier
}

func NewReconciliationService(repository *Repository, config JobConfig, workerID string) *ReconciliationService {
	return &ReconciliationService{
		repository:           repository,
		providerTimeout:      config.ProviderTimeout,
		pendingCheckInterval: config.PendingCheckInterval,
		retryBaseInterval:    config.RetryBaseInterval,
		maxRetryInterval:     config.MaxRetryInterval,
		workerID:             strings.TrimSpace(workerID),
		now:                  func() time.Time { return time.Now().UTC() },
	}
}

func (s *ReconciliationService) WithNotifier(notifier reconciliationNotifier) *ReconciliationService {
	if s == nil {
		return nil
	}
	s.notifier = notifier
	return s
}

func (s *ReconciliationService) Process(ctx context.Context, provider relaysenderid.Provider, claim RegistrationClaim) error {
	if claim.ProviderSubmittedAt == nil && !strings.EqualFold(claim.ProviderStatus, providerStatusSubmissionUnknown) {
		return s.submit(ctx, provider, claim)
	}
	return s.checkStatus(ctx, provider, claim)
}

func (s *ReconciliationService) submit(ctx context.Context, provider relaysenderid.Provider, claim RegistrationClaim) error {
	record, err := s.repository.Get(ctx, claim.ID, claim.TeamID)
	if err != nil {
		return s.recordFailure(ctx, claim, providerStatusSubmissionUnknown, err)
	}

	providerCtx, cancel := context.WithTimeout(ctx, s.providerTimeout)
	response, err := provider.CreateSenderID(providerCtx, relaysenderid.CreateRequest{
		Name:        claim.Name,
		CountryCode: claim.CountryCode,
		Purpose:     record.Purpose,
	})
	cancel()
	if err != nil {
		providerStatus := providerStatusSubmissionFailed
		if !definitiveProviderError(err) {
			providerStatus = providerStatusSubmissionUnknown
		}
		return s.recordFailure(ctx, claim, providerStatus, err)
	}
	if err := validateCreateResponse(provider, claim, response); err != nil {
		return s.recordFailure(ctx, claim, providerStatusSubmissionUnknown, err)
	}

	providerStatus := strings.TrimSpace(response.ProviderStatus)
	if providerStatus == "" {
		providerStatus = string(response.Status)
	}
	switch response.Status {
	case relaysenderid.StatusPending:
		return s.repository.CompleteSubmission(
			ctx,
			claim.ID,
			s.workerID,
			providerStatus,
			s.now().Add(s.pendingCheckInterval),
		)
	case relaysenderid.StatusApproved, relaysenderid.StatusRejected, relaysenderid.StatusSuspended:
		return s.completeStatus(ctx, claim, relaysenderid.StatusResult{
			Provider:          response.Provider,
			Name:              response.Name,
			ProviderReference: response.ProviderReference,
			Status:            response.Status,
			ProviderStatus:    providerStatus,
			ProviderCode:      response.ProviderCode,
		})
	default:
		return s.recordFailure(
			ctx,
			claim,
			providerStatusSubmissionUnknown,
			fmt.Errorf("provider returned unknown Sender ID creation status %q", response.Status),
		)
	}
}

func (s *ReconciliationService) checkStatus(ctx context.Context, provider relaysenderid.Provider, claim RegistrationClaim) error {
	providerCtx, cancel := context.WithTimeout(ctx, s.providerTimeout)
	response, err := provider.CheckSenderIDStatus(providerCtx, relaysenderid.StatusRequest{
		Provider: claim.Provider,
		Name:     claim.Name,
	})
	cancel()
	if err != nil {
		providerStatus := claim.ProviderStatus
		if providerStatus == "" {
			providerStatus = string(relaysenderid.StatusUnknown)
		}
		return s.recordFailure(ctx, claim, providerStatus, err)
	}
	if err := validateStatusResponse(provider, claim, response); err != nil {
		return s.recordFailure(ctx, claim, string(relaysenderid.StatusUnknown), err)
	}
	return s.completeStatus(ctx, claim, response)
}

func (s *ReconciliationService) completeStatus(ctx context.Context, claim RegistrationClaim, response relaysenderid.StatusResult) error {
	providerStatus := strings.TrimSpace(response.ProviderStatus)
	if providerStatus == "" {
		providerStatus = string(response.Status)
	}

	var rejectionReason *string
	nextCheckAt := s.now()
	switch response.Status {
	case relaysenderid.StatusPending:
		nextCheckAt = nextCheckAt.Add(s.pendingCheckInterval)
	case relaysenderid.StatusApproved:
	case relaysenderid.StatusRejected:
		reason := "Sender ID was rejected by " + response.Provider
		rejectionReason = &reason
	case relaysenderid.StatusSuspended:
	default:
		return s.recordFailure(
			ctx,
			claim,
			providerStatus,
			fmt.Errorf("provider returned unknown Sender ID status %q", response.Status),
		)
	}

	status := string(response.Status)
	whitelisted := response.Status == relaysenderid.StatusApproved
	if err := s.repository.CompleteStatus(
		ctx,
		claim.ID,
		s.workerID,
		status,
		providerStatus,
		whitelisted,
		rejectionReason,
		nextCheckAt,
	); err != nil {
		return err
	}
	s.notify(ctx, claim, status, rejectionReason)
	return nil
}

func (s *ReconciliationService) recordFailure(ctx context.Context, claim RegistrationClaim, providerStatus string, cause error) error {
	nextCheckAt := s.now().Add(s.retryDelay(claim.Attempt))
	recordErr := s.repository.RecordProviderFailure(
		ctx,
		claim.ID,
		s.workerID,
		providerStatus,
		cause,
		nextCheckAt,
	)
	return errors.Join(cause, recordErr)
}

func (s *ReconciliationService) retryDelay(attempt int32) time.Duration {
	delay := s.retryBaseInterval
	for current := int32(1); current < attempt && delay < s.maxRetryInterval; current++ {
		if delay > s.maxRetryInterval/2 {
			return s.maxRetryInterval
		}
		delay *= 2
	}
	if delay > s.maxRetryInterval {
		return s.maxRetryInterval
	}
	return delay
}

func (s *ReconciliationService) notify(ctx context.Context, claim RegistrationClaim, status string, reason *string) {
	if s.notifier == nil || !notifiableStatus(status) {
		return
	}
	recipients, err := s.repository.ListNotificationRecipients(ctx, claim.TeamID)
	if err != nil {
		return
	}
	reasonText := ""
	if reason != nil {
		reasonText = *reason
	}
	for _, recipient := range recipients {
		_ = s.notifier.SendSenderIDStatus(ctx, systemmail.SendSenderIDStatusInput{
			ToEmail:  recipient.Email,
			Name:     recipient.Name,
			SenderID: claim.Name,
			Status:   status,
			Reason:   reasonText,
		})
	}
}

func notifiableStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case StatusApproved, StatusRejected, StatusSuspended:
		return true
	default:
		return false
	}
}

func validateCreateResponse(provider relaysenderid.Provider, claim RegistrationClaim, response relaysenderid.CreateResult) error {
	if !strings.EqualFold(strings.TrimSpace(response.Provider), strings.TrimSpace(provider.Name())) {
		return fmt.Errorf("sender ID provider response ID %q does not match %q", response.Provider, provider.Name())
	}
	if !strings.EqualFold(strings.TrimSpace(response.Name), strings.TrimSpace(claim.Name)) {
		return fmt.Errorf("sender ID provider response name %q does not match %q", response.Name, claim.Name)
	}
	return nil
}

func validateStatusResponse(provider relaysenderid.Provider, claim RegistrationClaim, response relaysenderid.StatusResult) error {
	if !strings.EqualFold(strings.TrimSpace(response.Provider), strings.TrimSpace(provider.Name())) {
		return fmt.Errorf("sender ID provider response ID %q does not match %q", response.Provider, provider.Name())
	}
	if !strings.EqualFold(strings.TrimSpace(response.Name), strings.TrimSpace(claim.Name)) {
		return fmt.Errorf("sender ID provider response name %q does not match %q", response.Name, claim.Name)
	}
	return nil
}

func definitiveProviderError(err error) bool {
	var definitive safeFallbackError
	return errors.As(err, &definitive) && definitive.SafeToFallback()
}

func (s *Service) validateCreate(req CreateRequest) (string, string, string, *string, error) {
	name := relaysenderid.NormalizeName(req.Name)
	countryCode := relaysenderid.NormalizeCountryCode(req.CountryCode)
	purpose := strings.TrimSpace(req.Purpose)
	provider := normalizeOptional(req.Provider)

	if err := relaysenderid.ValidateName(name); err != nil {
		return "", "", "", nil, apperrors.NewBadRequest(err.Error())
	}
	if err := relaysenderid.ValidateCountryCode(countryCode); err != nil {
		return "", "", "", nil, apperrors.NewBadRequest("Country code must be a valid ISO 3166-1 alpha-2 code")
	}
	if purpose == "" {
		return "", "", "", nil, apperrors.NewBadRequest("Sender ID purpose is required")
	}
	if len(purpose) > maxPurposeLength {
		return "", "", "", nil, apperrors.NewBadRequest("Sender ID purpose must be at most 500 characters")
	}
	if provider != nil && len(*provider) > maxProviderLength {
		return "", "", "", nil, apperrors.NewBadRequest("Sender ID provider must be at most 120 characters")
	}

	if provider == nil {
		if len(s.providers) == 0 {
			return "", "", "", nil, apperrors.NewServiceUnavailable("Sender ID provider is not configured", nil)
		}
		value := s.providers[0]
		provider = &value
	} else if len(s.providers) != 0 && !containsProvider(s.providers, *provider) {
		return "", "", "", nil, apperrors.NewBadRequest("Sender ID provider is not configured")
	}
	return name, countryCode, purpose, provider, nil
}

func containsProvider(providers []string, provider string) bool {
	for _, value := range providers {
		if strings.EqualFold(value, provider) {
			return true
		}
	}
	return false
}

func normalizeOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.ToLower(strings.TrimSpace(*value))
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func requireTenantPermission(ctx context.Context, permission authz.Permission) (authz.Access, error) {
	tenantContext, decision := authz.ResolveAccess(ctx, permission)
	if !decision.Allowed {
		return authz.Access{}, apperrors.NewForbidden(decision.Reason)
	}
	return tenantContext, nil
}
