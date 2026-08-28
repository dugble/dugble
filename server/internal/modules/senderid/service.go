package senderid

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/dugble/dugble/server/internal/authz"
	platformsenderid "github.com/dugble/dugble/server/internal/platform/senderid"
	"github.com/dugble/dugble/server/internal/platform/systemmail"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

const (
	maxPurposeLength  = 500
	defaultProviderID = platformsenderid.ProviderMoolre
)

var countryCodePattern = regexp.MustCompile(`^[A-Z]{2}$`)

type Service struct{ repository *Repository }

func NewService(repository *Repository) *Service { return &Service{repository: repository} }

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
	name, countryCode, purpose, provider, err := validateCreate(req)
	if err != nil {
		return SenderID{}, err
	}
	senderID, err := s.repository.Create(ctx, tenantContext.Scope.TeamID, name, countryCode, purpose, provider, tenantContext.Actor.UserIDPtr())
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
		repository: repository, providerTimeout: config.ProviderTimeout,
		pendingCheckInterval: config.PendingCheckInterval,
		retryBaseInterval:    config.RetryBaseInterval, maxRetryInterval: config.MaxRetryInterval,
		workerID: strings.TrimSpace(workerID), now: func() time.Time { return time.Now().UTC() },
	}
}

func (s *ReconciliationService) WithNotifier(notifier reconciliationNotifier) *ReconciliationService {
	s.notifier = notifier
	return s
}

func (s *ReconciliationService) Process(ctx context.Context, provider platformsenderid.Provider, claim RegistrationClaim) error {
	if claim.ProviderSubmittedAt == nil && !strings.EqualFold(claim.ProviderStatus, providerStatusSubmissionUnknown) {
		return s.submit(ctx, provider, claim)
	}
	return s.checkStatus(ctx, provider, claim)
}

func (s *ReconciliationService) submit(ctx context.Context, provider platformsenderid.Provider, claim RegistrationClaim) error {
	providerCtx, cancel := context.WithTimeout(ctx, s.providerTimeout)
	response, err := provider.Create(providerCtx, platformsenderid.CreateRequest{SenderID: claim.Name})
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
	switch response.Status {
	case platformsenderid.StatusPending:
		return s.repository.CompleteSubmission(ctx, claim.ID, s.workerID, response.Status, s.now().Add(s.pendingCheckInterval))
	case platformsenderid.StatusApproved, platformsenderid.StatusRejected, platformsenderid.StatusSuspended:
		return s.completeStatus(ctx, claim, &platformsenderid.StatusResponse{ProviderID: response.ProviderID, SenderID: response.SenderID, Status: response.Status, ProviderStatus: response.Status})
	default:
		return s.recordFailure(ctx, claim, providerStatusSubmissionUnknown, fmt.Errorf("provider returned unknown Sender ID creation status %q", response.Status))
	}
}

func (s *ReconciliationService) checkStatus(ctx context.Context, provider platformsenderid.Provider, claim RegistrationClaim) error {
	providerCtx, cancel := context.WithTimeout(ctx, s.providerTimeout)
	response, err := provider.CheckStatus(providerCtx, claim.Name)
	cancel()
	if err != nil {
		providerStatus := claim.ProviderStatus
		if providerStatus == "" {
			providerStatus = platformsenderid.StatusUnknown
		}
		return s.recordFailure(ctx, claim, providerStatus, err)
	}
	if err := validateStatusResponse(provider, claim, response); err != nil {
		return s.recordFailure(ctx, claim, platformsenderid.StatusUnknown, err)
	}
	return s.completeStatus(ctx, claim, response)
}

func (s *ReconciliationService) completeStatus(ctx context.Context, claim RegistrationClaim, response *platformsenderid.StatusResponse) error {
	var rejectionReason *string
	nextCheckAt := s.now()
	switch response.Status {
	case platformsenderid.StatusPending:
		nextCheckAt = nextCheckAt.Add(s.pendingCheckInterval)
	case platformsenderid.StatusApproved:
	case platformsenderid.StatusRejected:
		reason := "Sender ID was rejected by " + response.ProviderID
		rejectionReason = &reason
	case platformsenderid.StatusSuspended:
	default:
		return s.recordFailure(ctx, claim, response.ProviderStatus, fmt.Errorf("provider returned unknown Sender ID status %q", response.Status))
	}
	if err := s.repository.CompleteStatus(ctx, claim.ID, s.workerID, response.Status, response.ProviderStatus, response.Whitelisted, rejectionReason, nextCheckAt); err != nil {
		return err
	}
	s.notify(ctx, claim, response.Status, rejectionReason)
	return nil
}

func (s *ReconciliationService) recordFailure(ctx context.Context, claim RegistrationClaim, providerStatus string, cause error) error {
	nextCheckAt := s.now().Add(s.retryDelay(claim.Attempt))
	recordErr := s.repository.RecordProviderFailure(ctx, claim.ID, s.workerID, providerStatus, cause, nextCheckAt)
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
		_ = s.notifier.SendSenderIDStatus(ctx, systemmail.SendSenderIDStatusInput{ToEmail: recipient.Email, Name: recipient.Name, SenderID: claim.Name, Status: status, Reason: reasonText})
	}
}

func notifiableStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case platformsenderid.StatusApproved, platformsenderid.StatusRejected, platformsenderid.StatusSuspended:
		return true
	default:
		return false
	}
}

func validateCreateResponse(provider platformsenderid.Provider, claim RegistrationClaim, response *platformsenderid.CreateResponse) error {
	if response == nil {
		return errors.New("sender ID provider returned an empty creation response")
	}
	if !strings.EqualFold(strings.TrimSpace(response.ProviderID), strings.TrimSpace(provider.ID())) {
		return fmt.Errorf("sender ID provider response ID %q does not match %q", response.ProviderID, provider.ID())
	}
	if !strings.EqualFold(strings.TrimSpace(response.SenderID), strings.TrimSpace(claim.Name)) {
		return fmt.Errorf("sender ID provider response name %q does not match %q", response.SenderID, claim.Name)
	}
	return nil
}

func validateStatusResponse(provider platformsenderid.Provider, claim RegistrationClaim, response *platformsenderid.StatusResponse) error {
	if response == nil {
		return errors.New("sender ID provider returned an empty status response")
	}
	if !strings.EqualFold(strings.TrimSpace(response.ProviderID), strings.TrimSpace(provider.ID())) {
		return fmt.Errorf("sender ID provider response ID %q does not match %q", response.ProviderID, provider.ID())
	}
	if !strings.EqualFold(strings.TrimSpace(response.SenderID), strings.TrimSpace(claim.Name)) {
		return fmt.Errorf("sender ID provider response name %q does not match %q", response.SenderID, claim.Name)
	}
	return nil
}

func definitiveProviderError(err error) bool {
	var definitive safeFallbackError
	return errors.As(err, &definitive) && definitive.SafeToFallback()
}

func validateCreate(req CreateRequest) (string, string, string, *string, error) {
	name := platformsenderid.NormalizeName(req.Name)
	countryCode := strings.ToUpper(strings.TrimSpace(req.CountryCode))
	purpose := strings.TrimSpace(req.Purpose)
	provider := defaultProviderID
	if err := platformsenderid.ValidateName(name); err != nil {
		return "", "", "", nil, apperrors.NewBadRequest(err.Error())
	}
	if !countryCodePattern.MatchString(countryCode) {
		return "", "", "", nil, apperrors.NewBadRequest("Country code must be a valid ISO 3166-1 alpha-2 code")
	}
	if purpose == "" {
		return "", "", "", nil, apperrors.NewBadRequest("Sender ID purpose is required")
	}
	if len(purpose) > maxPurposeLength {
		return "", "", "", nil, apperrors.NewBadRequest("Sender ID purpose must be at most 500 characters")
	}
	if countryCode != "GH" {
		return "", "", "", nil, apperrors.NewBadRequest("Sender ID registration is currently available only for Ghana")
	}
	return name, countryCode, purpose, &provider, nil
}

func requireTenantPermission(ctx context.Context, permission authz.Permission) (authz.Access, error) {
	tenantContext, decision := authz.ResolveAccess(ctx, permission)
	if !decision.Allowed {
		return authz.Access{}, apperrors.NewForbidden(decision.Reason)
	}
	return tenantContext, nil
}
