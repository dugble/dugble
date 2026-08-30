package broadcastexecution

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	broadcastmodule "github.com/dugble/dugble/server/internal/modules/broadcast"
	emailmodule "github.com/dugble/dugble/server/internal/modules/email"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

type processorTx struct {
	pgx.Tx
	committed  bool
	rolledBack bool
	children   []*processorTx
}

func (tx *processorTx) Begin(context.Context) (pgx.Tx, error) {
	child := &processorTx{}
	tx.children = append(tx.children, child)
	return child, nil
}

func (tx *processorTx) Commit(context.Context) error {
	tx.committed = true
	return nil
}

func (tx *processorTx) Rollback(context.Context) error {
	tx.rolledBack = true
	return nil
}

type processorRepository struct {
	recipients      []broadcastmodule.FanoutRecipient
	transactions    []*processorTx
	queuedMessageID *uuid.UUID
	retryAt         *time.Time
	retryCode       string
	failed          bool
	finalized       int
	exclusionReason string
}

func (repository *processorRepository) QueueNextDueScheduled(context.Context) (broadcastmodule.Broadcast, bool, error) {
	return broadcastmodule.Broadcast{}, false, nil
}

func (repository *processorRepository) MaterializeNextQueuedRecipients(context.Context) (broadcastmodule.MaterializationResult, bool, error) {
	return broadcastmodule.MaterializationResult{}, false, nil
}

func (repository *processorRepository) BeginFanoutTx(context.Context) (pgx.Tx, error) {
	tx := &processorTx{}
	repository.transactions = append(repository.transactions, tx)
	return tx, nil
}

func (repository *processorRepository) ClaimNextRecipientForFanoutTx(context.Context, pgx.Tx) (broadcastmodule.FanoutRecipient, bool, error) {
	if len(repository.recipients) == 0 {
		return broadcastmodule.FanoutRecipient{}, false, nil
	}
	recipient := repository.recipients[0]
	repository.recipients = repository.recipients[1:]
	return recipient, true, nil
}

func (repository *processorRepository) SetRecipientQueuedTx(_ context.Context, _ pgx.Tx, _ broadcastmodule.FanoutRecipient, messageID uuid.UUID) error {
	repository.queuedMessageID = &messageID
	return nil
}

func (repository *processorRepository) RecheckRecipientEligibilityTx(context.Context, pgx.Tx, broadcastmodule.FanoutRecipient) (string, bool, error) {
	return repository.exclusionReason, repository.exclusionReason != "", nil
}

func (repository *processorRepository) RetryRecipientFanoutTx(_ context.Context, _ pgx.Tx, _ broadcastmodule.FanoutRecipient, next time.Time, code, _ string) error {
	repository.retryAt = &next
	repository.retryCode = code
	return nil
}

func (repository *processorRepository) FailRecipientFanoutTx(context.Context, pgx.Tx, broadcastmodule.FanoutRecipient, string, string) error {
	repository.failed = true
	return nil
}

func (repository *processorRepository) FinalizeBroadcastFanoutTx(context.Context, pgx.Tx, uuid.UUID, uuid.UUID) (broadcastmodule.Broadcast, error) {
	repository.finalized++
	return broadcastmodule.Broadcast{Status: broadcastmodule.StatusQueued}, nil
}

type processorEmail struct {
	result       emailmodule.QueuedMessage
	err          error
	request      emailmodule.SendRequest
	enqueueCalls int
	observeCalls int
}

func (email *processorEmail) EnqueueMarketingTx(_ context.Context, _ pgx.Tx, _ uuid.UUID, request emailmodule.SendRequest) (emailmodule.QueuedMessage, error) {
	email.enqueueCalls++
	email.request = request
	return email.result, email.err
}

func (email *processorEmail) ObserveCommitted(context.Context, emailmodule.QueuedMessage) {
	email.observeCalls++
}

type fixedClock struct{ value time.Time }

func (clock fixedClock) Now() time.Time { return clock.value }

type testUnsubscribeLinker struct{}

func (testUnsubscribeLinker) Link(recipientID uuid.UUID) (string, error) {
	return "https://api.dugble.test/unsubscribe?token=" + recipientID.String(), nil
}

func testRecipient() broadcastmodule.FanoutRecipient {
	firstName := "Ada"
	lastName := "Lovelace"
	fromEmail := "hello@example.com"
	fromName := "Dugble"
	text := "Hello {{{FIRST_NAME}}}, your plan is {{{plan}}}."
	return broadcastmodule.FanoutRecipient{
		ID:          uuid.New(),
		TeamID:      uuid.New(),
		BroadcastID: uuid.New(),
		Email:       "ada@example.com",
		FirstName:   &firstName,
		LastName:    &lastName,
		ContactSnapshot: map[string]any{
			"properties": map[string]any{"FIRST_NAME": "property", "plan": "pro"},
		},
		FromEmail:        &fromEmail,
		FromName:         &fromName,
		Subject:          "Hello {{{FIRST_NAME}}}",
		HTML:             "<p>Hello {{{FIRST_NAME}}}, plan {{{plan}}}</p><a href=\"{{{UNSUBSCRIBE_URL}}}\">unsubscribe</a>",
		Text:             &text,
		VariableBindings: map[string]any{"plan": "enterprise", "campaign": "august"},
	}
}

func TestResolveRecipientVariablesUsesDocumentedPrecedence(t *testing.T) {
	recipient := testRecipient()
	variables := resolveRecipientVariables(recipient)

	if variables["EMAIL"] != recipient.Email {
		t.Fatalf("EMAIL = %v, want %s", variables["EMAIL"], recipient.Email)
	}
	if variables["FIRST_NAME"] != "Ada" {
		t.Fatalf("FIRST_NAME = %v, want Ada", variables["FIRST_NAME"])
	}
	if variables["plan"] != "enterprise" {
		t.Fatalf("plan = %v, want enterprise", variables["plan"])
	}
	if variables["campaign"] != "august" {
		t.Fatalf("campaign = %v, want august", variables["campaign"])
	}
}

func TestRetryPolicyBackoffIsBounded(t *testing.T) {
	policy := RetryPolicy{MaxAttempts: 8, BaseDelay: time.Minute, MaxDelay: 5 * time.Minute}
	if delay := policy.delay(0); delay != time.Minute {
		t.Fatalf("first delay = %v, want 1m", delay)
	}
	if delay := policy.delay(4); delay != 5*time.Minute {
		t.Fatalf("bounded delay = %v, want 5m", delay)
	}
}

func TestClassifyFanoutFailure(t *testing.T) {
	if failure := classifyFanoutFailure(apperrors.NewBadRequest("invalid rendered message")); failure.retryable {
		t.Fatal("bad request should be terminal")
	}
	if failure := classifyFanoutFailure(apperrors.NewServiceUnavailable("pricing unavailable", errors.New("offline"))); !failure.retryable {
		t.Fatal("service unavailable should be retryable")
	}
}

func TestProcessorQueuesOwnedBroadcastContent(t *testing.T) {
	recipient := testRecipient()
	messageID := uuid.New()
	repository := &processorRepository{recipients: []broadcastmodule.FanoutRecipient{recipient}}
	email := &processorEmail{result: emailmodule.QueuedMessage{Message: emailmodule.Message{ID: messageID.String()}}}
	processor := NewProcessor(repository, email, testUnsubscribeLinker{})

	if err := processor.ProcessBatch(context.Background(), 10); err != nil {
		t.Fatalf("ProcessBatch returned error: %v", err)
	}
	if repository.queuedMessageID == nil || *repository.queuedMessageID != messageID {
		t.Fatalf("queued message id = %v, want %s", repository.queuedMessageID, messageID)
	}
	if repository.finalized != 1 {
		t.Fatalf("finalized = %d, want 1", repository.finalized)
	}
	if email.observeCalls != 1 {
		t.Fatalf("observe calls = %d, want 1", email.observeCalls)
	}
	if email.request.Subject != "Hello Ada" {
		t.Fatalf("subject = %q, want Hello Ada", email.request.Subject)
	}
	if email.request.HTML == recipient.HTML || email.request.HTML == "" {
		t.Fatalf("html was not rendered: %q", email.request.HTML)
	}
	if email.request.Headers["List-Unsubscribe"] == "" {
		t.Fatal("List-Unsubscribe header is missing")
	}
	if email.request.Headers["List-Unsubscribe-Post"] != "List-Unsubscribe=One-Click" {
		t.Fatalf("List-Unsubscribe-Post = %q", email.request.Headers["List-Unsubscribe-Post"])
	}
	if len(repository.transactions) < 1 || !repository.transactions[0].committed {
		t.Fatal("recipient transaction was not committed")
	}
}

func TestProcessorExcludesRecipientWhoUnsubscribedAfterMaterialization(t *testing.T) {
	repository := &processorRepository{
		recipients:      []broadcastmodule.FanoutRecipient{testRecipient()},
		exclusionReason: "global_unsubscribe",
	}
	email := &processorEmail{}
	processor := NewProcessor(repository, email, testUnsubscribeLinker{})

	if err := processor.ProcessBatch(context.Background(), 10); err != nil {
		t.Fatalf("ProcessBatch returned error: %v", err)
	}
	if email.enqueueCalls != 0 {
		t.Fatalf("email enqueue calls = %d, want 0", email.enqueueCalls)
	}
	if repository.finalized != 1 {
		t.Fatalf("finalized = %d, want 1", repository.finalized)
	}
	if len(repository.transactions) < 1 || !repository.transactions[0].committed {
		t.Fatal("excluded recipient transaction was not committed")
	}
}

func TestProcessorRecordsTerminalRenderFailure(t *testing.T) {
	recipient := testRecipient()
	recipient.Subject = "Hello {{{MISSING}}}"
	repository := &processorRepository{recipients: []broadcastmodule.FanoutRecipient{recipient}}
	email := &processorEmail{}
	processor := NewProcessor(repository, email, testUnsubscribeLinker{})

	if err := processor.ProcessBatch(context.Background(), 10); err != nil {
		t.Fatalf("ProcessBatch returned error: %v", err)
	}
	if !repository.failed {
		t.Fatal("terminal render failure was not recorded")
	}
	if repository.retryAt != nil {
		t.Fatal("terminal render failure should not be retried")
	}
	if email.enqueueCalls != 0 {
		t.Fatalf("email enqueue calls = %d, want 0", email.enqueueCalls)
	}
}

func TestProcessorSchedulesRetryAfterTransientEmailFailure(t *testing.T) {
	now := time.Date(2026, time.August, 7, 12, 0, 0, 0, time.UTC)
	repository := &processorRepository{recipients: []broadcastmodule.FanoutRecipient{testRecipient()}}
	email := &processorEmail{err: apperrors.NewServiceUnavailable("Email pricing is unavailable", errors.New("offline"))}
	processor := NewProcessor(repository, email, testUnsubscribeLinker{}, fixedClock{value: now})

	if err := processor.ProcessBatch(context.Background(), 10); err != nil {
		t.Fatalf("ProcessBatch returned error: %v", err)
	}
	if repository.retryAt == nil || !repository.retryAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("retry at = %v, want %v", repository.retryAt, now.Add(time.Minute))
	}
	if repository.failed {
		t.Fatal("transient email failure should not be terminal")
	}
	if len(repository.transactions) == 0 || len(repository.transactions[0].children) == 0 || !repository.transactions[0].children[0].rolledBack {
		t.Fatal("email savepoint was not rolled back")
	}
}
