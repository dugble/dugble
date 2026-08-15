package domainreconciliation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	domainmodule "github.com/dugble/dugble/server/internal/modules/domain"
	"github.com/dugble/dugble/server/internal/platform/systemmail"
)

type notifier interface {
	SendSenderDomainStatus(context.Context, systemmail.SendSenderDomainStatusInput) error
}

type Processor struct {
	repository repository
	checker    checker
	config     Config
	workerID   string
	now        func() time.Time
	notifier   notifier
}

func (p *Processor) WithNotifier(notifier notifier) *Processor {
	p.notifier = notifier
	return p
}

func NewProcessor(repository repository, checker checker, config Config, workerID string) *Processor {
	if config.PendingCheckInterval <= 0 {
		config.PendingCheckInterval = 30 * time.Second
	}
	return &Processor{
		repository: repository,
		checker:    checker,
		config:     config,
		workerID:   workerID,
		now:        func() time.Time { return time.Now().UTC() },
	}
}

func (p *Processor) Process(ctx context.Context, claim domainmodule.ReconciliationClaim) error {
	if p == nil || p.repository == nil || p.checker == nil {
		return ErrConsumerNotConfigured
	}
	id, err := uuid.Parse(claim.Domain.ID)
	if err != nil {
		return fmt.Errorf("parse sender domain id: %w", err)
	}
	checkCtx, cancel := context.WithTimeout(ctx, p.config.CheckTimeout)
	defer cancel()
	result, checkErr := p.checker.Check(checkCtx, claim.Domain)
	if claim.Domain.Status == domainmodule.StatusVerified {
		return p.completeHealthCheck(ctx, id, claim.Domain, result, checkErr)
	}

	delay := nextVerificationDelay(result.Status, checkErr, claim.Attempt, id, p.config)
	nextCheckAt := p.now().Add(delay)
	if checkErr != nil {
		_, recordErr := p.repository.RecordReconciliationFailure(ctx, id, p.workerID, checkErr, nextCheckAt)
		return errors.Join(checkErr, recordErr)
	}

	updated, err := p.repository.CompleteReconciliation(ctx, id, p.workerID, result.Status, result.VerificationRecords, nextCheckAt)
	if err != nil {
		return err
	}
	if err := p.repository.ResetReconciliationAttempts(ctx, id); err != nil {
		return err
	}
	p.notify(ctx, claim.Domain, updated)
	return nil
}

func nextVerificationDelay(status string, checkErr error, attempt int32, id uuid.UUID, config Config) time.Duration {
	if checkErr != nil {
		return nextCheckDelay(max(attempt-1, 0), id)
	}
	if status == domainmodule.StatusVerified {
		return jitter(config.HealthCheckInterval, id)
	}
	return jitter(config.PendingCheckInterval, id)
}

func (p *Processor) completeHealthCheck(ctx context.Context, id uuid.UUID, previous domainmodule.SenderDomain, result domainmodule.ReconciliationResult, checkErr error) error {
	if checkErr == nil && result.Status == domainmodule.StatusVerified {
		_, err := p.repository.CompleteHealthCheck(ctx, id, p.workerID, p.now().Add(jitter(p.config.HealthCheckInterval, id)))
		return err
	}
	if checkErr == nil {
		checkErr = errors.New("sender domain verification checks no longer pass")
	}
	updated, recordErr := p.repository.RecordHealthFailure(
		ctx,
		id,
		p.workerID,
		checkErr,
		p.config.HealthFailureThreshold,
		p.now().Add(jitter(p.config.HealthRetryInterval, id)),
	)
	if recordErr == nil {
		reason := checkErr.Error()
		updated.FailureReason = &reason
		p.notify(ctx, previous, updated)
	}
	return errors.Join(checkErr, recordErr)
}

func (p *Processor) notify(ctx context.Context, previous, updated domainmodule.SenderDomain) {
	status := domainmodule.NotificationStatus(previous, updated)
	if status == "" || p.notifier == nil {
		return
	}
	teamID, err := uuid.Parse(updated.TeamID)
	if err != nil {
		return
	}
	recipients, err := p.repository.ListNotificationRecipients(ctx, teamID)
	if err != nil {
		return
	}
	reason := ""
	if updated.FailureReason != nil {
		reason = *updated.FailureReason
	}
	for _, recipient := range recipients {
		_ = p.notifier.SendSenderDomainStatus(ctx, systemmail.SendSenderDomainStatusInput{ToEmail: recipient.Email, Name: recipient.Name, Domain: updated.Domain, Status: status, Reason: reason})
	}
}
