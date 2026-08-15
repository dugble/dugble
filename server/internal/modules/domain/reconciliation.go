package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/dugble/dugble/server/internal/platform/systemmail"
)

type reconciliationChecker interface {
	Check(context.Context, SenderDomain) (ReconciliationResult, error)
}

type ReconciliationService struct {
	repository *Repository
	checker    reconciliationChecker
	config     JobConfig
	workerID   string
	now        func() time.Time
	notifier   statusNotifier
}

func NewReconciliationService(repository *Repository, checker reconciliationChecker, config JobConfig, workerID string) *ReconciliationService {
	if config.PendingCheckInterval <= 0 {
		config.PendingCheckInterval = 30 * time.Second
	}
	return &ReconciliationService{
		repository: repository,
		checker:    checker,
		config:     config,
		workerID:   workerID,
		now:        func() time.Time { return time.Now().UTC() },
	}
}

func (s *ReconciliationService) WithNotifier(notifier statusNotifier) *ReconciliationService {
	s.notifier = notifier
	return s
}

func (s *ReconciliationService) Process(ctx context.Context, claim ReconciliationClaim) error {
	if s == nil || s.repository == nil || s.checker == nil {
		return ErrJobNotConfigured
	}
	id, err := uuid.Parse(claim.Domain.ID)
	if err != nil {
		return fmt.Errorf("parse sender domain id: %w", err)
	}
	checkCtx, cancel := context.WithTimeout(ctx, s.config.CheckTimeout)
	defer cancel()
	result, checkErr := s.checker.Check(checkCtx, claim.Domain)
	if claim.Domain.Status == StatusVerified {
		return s.completeHealthCheck(ctx, id, claim.Domain, result, checkErr)
	}

	delay := nextVerificationDelay(result.Status, checkErr, claim.Attempt, id, s.config)
	nextCheckAt := s.now().Add(delay)
	if checkErr != nil {
		_, recordErr := s.repository.RecordReconciliationFailure(ctx, id, s.workerID, checkErr, nextCheckAt)
		return errors.Join(checkErr, recordErr)
	}

	updated, err := s.repository.CompleteReconciliation(ctx, id, s.workerID, result.Status, result.VerificationRecords, nextCheckAt)
	if err != nil {
		return err
	}
	if err := s.repository.ResetReconciliationAttempts(ctx, id); err != nil {
		return err
	}
	s.notify(ctx, claim.Domain, updated)
	return nil
}

func (s *ReconciliationService) completeHealthCheck(ctx context.Context, id uuid.UUID, previous SenderDomain, result ReconciliationResult, checkErr error) error {
	if checkErr == nil && result.Status == StatusVerified {
		_, err := s.repository.CompleteHealthCheck(ctx, id, s.workerID, s.now().Add(jitter(s.config.HealthCheckInterval, id)))
		return err
	}
	if checkErr == nil {
		checkErr = errors.New(manualHealthFailureReason)
	}
	updated, recordErr := s.repository.RecordHealthFailure(
		ctx,
		id,
		s.workerID,
		checkErr,
		s.config.HealthFailureThreshold,
		s.now().Add(jitter(s.config.HealthRetryInterval, id)),
	)
	if recordErr == nil {
		reason := checkErr.Error()
		updated.FailureReason = &reason
		s.notify(ctx, previous, updated)
	}
	return errors.Join(checkErr, recordErr)
}

func (s *ReconciliationService) notify(ctx context.Context, previous, updated SenderDomain) {
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
		_ = s.notifier.SendSenderDomainStatus(ctx, systemmail.SendSenderDomainStatusInput{
			ToEmail: recipient.Email,
			Name:    recipient.Name,
			Domain:  updated.Domain,
			Status:  status,
			Reason:  reason,
		})
	}
}
