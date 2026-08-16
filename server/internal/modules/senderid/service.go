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
	provider "github.com/dugble/dugble/server/internal/providers"
	"github.com/dugble/dugble/server/internal/platform/systemmail"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

const (
	maxPurposeLength  = 500
	maxProviderLength = 120
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
	name, countryCode, purpose, providerName, err := validateCreate(req)
	if err != nil {
		return SenderID{}, err
	}
	senderID, err := s.repository.Create(ctx, tenantContext.Scope.TeamID, name, countryCode, purpose, providerName, tenantContext.Actor.UserIDPtr())
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

type reconciliationProvider interface {
	provider.SenderIDCreator
	provider.SenderIDStatusChecker
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

func (s *ReconciliationService) Process(ctx context.Context, upstream reconciliationProvider, claim RegistrationClaim) error {
	if claim.ProviderSubmittedAt == nil && !strings.EqualFold(claim.ProviderStatus, providerStatusSubmissionUnknown) {
		return s.submit(ctx, upstream, claim)
	}
	return s.checkStatus(ctx, upstream, claim)
}

func (s *ReconciliationService) submit(ctx context.Context, upstream reconciliationProvider, claim RegistrationClaim) error {
	providerCtx, cancel := context.WithTimeout(ctx, s.providerTimeout)
	response, err := upstream.CreateSenderID(providerCtx, provider.CreateSenderIDRequest{SenderID: claim.Name})
	cancel()
	if err != nil {
		providerStatus := providerStatusSubmissionFailed
		if !definitiveProviderError(err) {
			providerStatus = providerStatusSubmissionUnknown
		}
		return s.recordFailure(ctx, claim, providerStatus, err)
	}
	if err := validateCreateResponse(claim, response); err != nil {
		return s.recordFailure(ctx, claim, providerStatusSubmissionUnknown, err)
	}
	status := normalizedSenderIDStatus(response.Status, "")
	switch status {
	case StatusPending:
		return s.repository.CompleteSubmission(ctx, claim.ID, s.workerID, status, s.now().Add(s.pendingCheckInterval))
	case StatusApproved, StatusRejected, StatusSuspended:
		return s.completeStatus(ctx, claim, upstream.Name(), provider.SenderIDStatusResult{
			SenderID:          response.SenderID,
			ProviderReference: response.ProviderReference,
			Status:            response.Status,
			ProviderStatus:    status,
		})
	default:
		return s.recordFailure(ctx, claim, providerStatusSubmissionUnknown, fmt.Errorf("provider returned unknown Sender ID creation status %q", response.Status))
	}
}

func (s *ReconciliationService) checkStatus(ctx context.Context, upstream reconciliationProvider, claim RegistrationClaim) error {
	providerCtx, cancel := context.WithTimeout(ctx, s.providerTimeout)
	response, err := upstream.CheckSenderIDStatus(providerCtx, provider.SenderIDStatusRequest{SenderID: claim.Name})
	cancel()
	if err != nil {
		providerStatus := claim.ProviderStatus
		if providerStatus == "" {
			providerStatus = string(provider.SenderIDUnknown)
		}
		return s.recordFailure(ctx, claim, providerStatus, err)
	}
	if err := validateStatusResponse(claim, response); err != nil {
		return s.recordFailure(ctx, claim, string(provider.SenderIDUnknown), err)
	}
	return s.completeStatus(ctx, claim, upstream.Name(), response)
}

func (s *ReconciliationService) completeStatus(ctx context.Context, claim RegistrationClaim, providerName string, response provider.SenderIDStatusResult) error {
	status := normalizedSenderIDStatus(response.Status, response.ProviderStatus)
	providerStatus := strings.TrimSpace(response.ProviderStatus)
	if providerStatus == "" {
		providerStatus = status
	}
	var rejectionReason *string
	nextCheckAt := s.now()
	switch status {
	case StatusPending:
		nextCheckAt = nextCheckAt.Add(s.pendingCheckInterval)
	case StatusApproved:
	case StatusRejected:
		reason := "Sender ID was rejected by " + strings.TrimSpace(providerName)
		rejectionReason = &reason
	case StatusSuspended:
	default:
		return s.recordFailure(ctx, claim, providerStatus, fmt.Errorf("provider returned unknown Sender ID status %q", response.Status))
	}
	if err := s.repository.CompleteStatus(ctx, claim.ID, s.workerID, status, providerStatus, response.Whitelisted, rejectionReason, nextCheckAt); err != nil {
		return err
	}
	s.notify(ctx, claim, status, rejectionReason)
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
	case StatusApproved, StatusRejected, StatusSuspended:
		return true
	default:
		return false
	}
}

func normalizedSenderIDStatus(status provider.SenderIDStatus, providerStatus string) string {
	switch strings.ToLower(strings.TrimSpace(providerStatus)) {
	case StatusPending:
		return StatusPending
	case StatusApproved:
		return StatusApproved
	case StatusRejected:
		return StatusRejected
	case StatusSuspended:
		return StatusSuspended
	}
	switch status {
	case provider.SenderIDPending:
		return StatusPending
	case provider.SenderIDActive:
		return StatusApproved
	case provider.SenderIDRejected:
		return StatusRejected
	default:
		return ""
	}
}

func validateCreateResponse(claim RegistrationClaim, response provider.CreateSenderIDResult) error {
	if !strings.EqualFold(strings.TrimSpace(response.SenderID), strings.TrimSpace(claim.Name)) {
		return fmt.Errorf("sender ID provider response name %q does not match %q", response.SenderID, claim.Name)
	}
	return nil
}

func validateStatusResponse(claim RegistrationClaim, response provider.SenderIDStatusResult) error {
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
	name := normalizeName(req.Name)
	countryCode := strings.ToUpper(strings.TrimSpace(req.CountryCode))
	purpose := strings.TrimSpace(req.Purpose)
	providerName := normalizeOptional(req.Provider)
	if err := validateName(name); err != nil {
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
	if providerName != nil && len(*providerName) > maxProviderLength {
		return "", "", "", nil, apperrors.NewBadRequest("Sender ID provider must be at most 120 characters")
	}
	if countryCode == "GH" {
		if providerName != nil && !strings.EqualFold(*providerName, ProviderMoolre) {
			return "", "", "", nil, apperrors.NewBadRequest("Ghana Sender IDs are registered through Moolre")
		}
		value := ProviderMoolre
		providerName = &value
	} else if providerName != nil && strings.EqualFold(*providerName, ProviderMoolre) {
		return "", "", "", nil, apperrors.NewBadRequest("Moolre Sender ID registration is currently available only for Ghana")
	}
	return name, countryCode, purpose, providerName, nil
}

func normalizeOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
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
