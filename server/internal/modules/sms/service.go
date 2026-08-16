package sms

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dugble/dugble/server/internal/authz"
	platformbilling "github.com/dugble/dugble/server/internal/billing/charge/usage"
	provider "github.com/dugble/dugble/server/internal/providers"
	relaysms "github.com/dugble/dugble/server/internal/relay/sms"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

type providerLookup interface {
	Provider(string) relaysms.Provider
}

type scheduledDeliveryQueue interface {
	EnqueueSMSDeliveryAtTx(context.Context, pgx.Tx, uuid.UUID, uuid.UUID, time.Time) error
	CancelSMSDeliveryTx(context.Context, pgx.Tx, uuid.UUID, uuid.UUID) error
	RescheduleSMSDeliveryTx(context.Context, pgx.Tx, uuid.UUID, uuid.UUID, time.Time) error
}

// DeliveryQueue enqueues durable SMS delivery work.
//
// Implementations should persist jobs durably before returning.
type DeliveryQueue interface {
	EnqueueSMSDelivery(ctx context.Context, messageID uuid.UUID, teamID uuid.UUID) error
	EnqueueSMSDeliveryTx(ctx context.Context, tx pgx.Tx, messageID uuid.UUID, teamID uuid.UUID) error
}

type Service struct {
	repository *Repository
	sender     providerLookup
	delivery   DeliveryQueue
	billing    platformbilling.SMSBilling
}

func NewService(repository *Repository, sender providerLookup, delivery DeliveryQueue, billing platformbilling.SMSBilling) *Service {
	return &Service{repository: repository, sender: sender, delivery: delivery, billing: billing}
}

func (s *Service) List(ctx context.Context, req ListRequest) ([]Message, error) {
	tenantContext, err := requireTenant(ctx, authz.PermissionSMSRead)
	if err != nil {
		return nil, err
	}
	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if req.Offset < 0 {
		req.Offset = 0
	}
	messages, err := s.repository.List(ctx, tenantContext.Scope.TeamID, limit, req.Offset)
	if err != nil {
		return nil, apperrors.NewInternal("Unable to list SMS messages", err)
	}
	return messages, nil
}

func (s *Service) Get(ctx context.Context, messageID string) (Message, error) {
	tenantContext, err := requireTenant(ctx, authz.PermissionSMSRead)
	if err != nil {
		return Message{}, err
	}
	parsedID, err := uuid.Parse(strings.TrimSpace(messageID))
	if err != nil {
		return Message{}, apperrors.NewBadRequest("SMS message id must be a valid UUID")
	}
	message, err := s.repository.Get(ctx, parsedID, tenantContext.Scope.TeamID)
	if err != nil {
		if errors.Is(err, ErrMessageNotFound) {
			return Message{}, apperrors.NewNotFound("SMS message not found")
		}
		return Message{}, apperrors.NewInternal("Unable to get SMS message", err)
	}
	return message, nil
}

func (s *Service) ListEvents(ctx context.Context, messageID string, limit int32) (EventListResponse, error) {
	tenantContext, err := requireTenant(ctx, authz.PermissionSMSRead)
	if err != nil {
		return EventListResponse{}, err
	}
	id, err := uuid.Parse(strings.TrimSpace(messageID))
	if err != nil {
		return EventListResponse{}, apperrors.NewBadRequest("SMS message id must be a valid UUID")
	}
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	if _, err := s.repository.Get(ctx, id, tenantContext.Scope.TeamID); errors.Is(err, ErrMessageNotFound) {
		return EventListResponse{}, apperrors.NewNotFound("SMS message not found")
	} else if err != nil {
		return EventListResponse{}, apperrors.NewInternal("Unable to get SMS message", err)
	}
	events, err := s.repository.ListEvents(ctx, tenantContext.Scope.TeamID, id, limit)
	if err != nil {
		return EventListResponse{}, apperrors.NewInternal("Unable to list SMS events", err)
	}
	return EventListResponse{Object: "list", Data: events}, nil
}

func (s *Service) Send(ctx context.Context, req SendRequest) (Message, error) {
	tenantContext, err := requireTenant(ctx, authz.PermissionSMSSend)
	if err != nil {
		return Message{}, err
	}
	if s.delivery == nil {
		return Message{}, apperrors.NewInternal("SMS delivery queue is not configured", nil)
	}
	if s.billing == nil {
		return Message{}, apperrors.NewInternal("SMS billing charge is not configured", nil)
	}

	normalized, err := validateSend(req)
	if err != nil {
		return Message{}, err
	}
	senderID, err := s.repository.FindApprovedSender(ctx, tenantContext.Scope.TeamID, normalized.From)
	if err != nil {
		return Message{}, apperrors.NewInternal("Unable to validate SMS sender ID", err)
	}
	if senderID == nil {
		return Message{}, apperrors.NewBadRequest("SMS sender ID must be approved before use")
	}

	segments := countSegments(normalized.Body)
	tx, err := s.repository.BeginTx(ctx)
	if err != nil {
		return Message{}, apperrors.NewInternal("Unable to begin SMS send transaction", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	txRepository := s.repository.WithTx(tx)

	created, err := txRepository.Create(ctx, createMessageParams{
		TeamID: tenantContext.Scope.TeamID, SenderID: senderID, To: normalized.To, From: normalized.From,
		Body: normalized.Body, Status: StatusQueued, Segments: segments,
		Metadata:           normalized.Metadata,
		ScheduledAt:        normalizedScheduledAt(normalized.ScheduledAt),
		DestinationCountry: normalized.DestinationCountry,
	})
	if err != nil {
		return Message{}, apperrors.NewInternal("Unable to create SMS message", err)
	}
	if created.ProviderMessageID != nil || created.Status != StatusQueued {
		return Message{}, apperrors.NewInternal("New SMS message is not in a billable queued state", nil)
	}

	messageID := uuid.MustParse(created.ID)
	charge, err := s.billing.ChargeSMS(ctx, tx, platformbilling.SMSChargeInput{
		TeamID: tenantContext.Scope.TeamID, MessageID: messageID,
		DestinationNumber: normalized.To, Segments: segments,
	})
	if err != nil {
		return Message{}, smsBillingError(err)
	}
	if err := enqueueSMSDelivery(ctx, s.delivery, tx, messageID, tenantContext.Scope.TeamID, created.ScheduledAt); err != nil {
		return Message{}, apperrors.NewInternal("Unable to enqueue SMS delivery", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Message{}, apperrors.NewInternal("Unable to commit SMS send transaction", err)
	}
	s.billing.ObserveCommittedCharge(ctx, platformbilling.CommittedCharge{
		Charge: charge, Channel: platformbilling.ChannelSMS,
		Settlement: platformbilling.SettlementAcceptedForDelivery,
		TeamID:     tenantContext.Scope.TeamID, MessageID: messageID,
	})

	return created, nil
}

func (s *Service) BatchSend(ctx context.Context, req BatchSendRequest) ([]Message, error) {
	if err := validateBatchSend(req); err != nil {
		return nil, err
	}
	tenantContext, err := requireTenant(ctx, authz.PermissionSMSSend)
	if err != nil {
		return nil, err
	}
	if s.delivery == nil {
		return nil, apperrors.NewInternal("SMS delivery queue is not configured", nil)
	}
	if s.billing == nil {
		return nil, apperrors.NewInternal("SMS billing charge is not configured", nil)
	}

	type preparedMessage struct {
		request  SendRequest
		senderID *uuid.UUID
		segments int32
	}
	prepared := make([]preparedMessage, len(req.Messages))
	senders := make(map[string]*uuid.UUID)
	for index, request := range req.Messages {
		normalized, err := validateSend(request)
		if err != nil {
			return nil, err
		}
		senderKey := strings.ToLower(normalized.From)
		senderID, exists := senders[senderKey]
		if !exists {
			senderID, err = s.repository.FindApprovedSender(ctx, tenantContext.Scope.TeamID, normalized.From)
			if err != nil {
				return nil, apperrors.NewInternal("Unable to validate SMS sender ID", err)
			}
			if senderID == nil {
				return nil, apperrors.NewBadRequest("SMS sender ID must be approved before use")
			}
			senders[senderKey] = senderID
		}
		prepared[index] = preparedMessage{request: normalized, senderID: senderID, segments: countSegments(normalized.Body)}
	}

	tx, err := s.repository.BeginTx(ctx)
	if err != nil {
		return nil, apperrors.NewInternal("Unable to begin SMS batch transaction", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	txRepository := s.repository.WithTx(tx)
	result := make([]Message, 0, len(prepared))
	committedCharges := make([]platformbilling.CommittedCharge, 0, len(prepared))
	for _, item := range prepared {
		created, err := txRepository.Create(ctx, createMessageParams{
			TeamID: item.requestTeamID(), SenderID: item.senderID, To: item.request.To, From: item.request.From,
			Body: item.request.Body, Status: StatusQueued, Segments: item.segments,
			Metadata:           item.request.Metadata,
			ScheduledAt:        normalizedScheduledAt(item.request.ScheduledAt),
			DestinationCountry: item.request.DestinationCountry,
		})
		_ = created
		_ = err
	}
	return result, nil
}
