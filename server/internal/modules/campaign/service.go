package smscampaign

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nyaruka/phonenumbers"

	"github.com/dugble/dugble/server/internal/authz"
	"github.com/dugble/dugble/server/internal/modules/sms"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
	"github.com/dugble/dugble/server/pkg/pgconv"
)

const maxBodyCharacters = 1600

type Service struct{ repository *Repository }

func NewService(repository *Repository) *Service { return &Service{repository: repository} }

func (s *Service) Create(ctx context.Context, req CreateRequest) (Campaign, error) {
	tc, err := require(ctx, authz.PermissionSMSSend)
	if err != nil {
		return Campaign{}, err
	}
	segmentID, senderID, req, err := validateCreate(req)
	if err != nil {
		return Campaign{}, err
	}
	value, err := s.repository.Create(ctx, tc.Scope.TeamID, segmentID, senderID, req)
	if errors.Is(err, ErrInvalidReference) {
		return Campaign{}, apperrors.NewBadRequest("Segment or sender ID was not found for this team")
	}
	if err != nil {
		return Campaign{}, apperrors.NewInternal("Unable to create SMS campaign", err)
	}
	return value, nil
}

func (s *Service) List(ctx context.Context, req ListRequest) ([]Campaign, error) {
	tc, err := require(ctx, authz.PermissionSMSRead)
	if err != nil {
		return nil, err
	}
	normalizeList(&req)
	values, err := s.repository.List(ctx, tc.Scope.TeamID, req)
	if err != nil {
		return nil, apperrors.NewInternal("Unable to list SMS campaigns", err)
	}
	return values, nil
}

func (s *Service) Get(ctx context.Context, id string) (Campaign, error) {
	tc, err := require(ctx, authz.PermissionSMSRead)
	if err != nil {
		return Campaign{}, err
	}
	campaignID, err := parseID(id, "Campaign id")
	if err != nil {
		return Campaign{}, err
	}
	value, err := s.repository.Get(ctx, tc.Scope.TeamID, campaignID)
	if errors.Is(err, ErrNotFound) {
		return Campaign{}, apperrors.NewNotFound("SMS campaign not found")
	}
	if err != nil {
		return Campaign{}, apperrors.NewInternal("Unable to get SMS campaign", err)
	}
	return value, nil
}

func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (Campaign, error) {
	tc, err := require(ctx, authz.PermissionSMSSend)
	if err != nil {
		return Campaign{}, err
	}
	campaignID, err := parseID(id, "Campaign id")
	if err != nil {
		return Campaign{}, err
	}
	if req.Revision <= 0 {
		return Campaign{}, apperrors.NewBadRequest("Revision is required")
	}
	current, err := s.repository.Get(ctx, tc.Scope.TeamID, campaignID)
	if errors.Is(err, ErrNotFound) {
		return Campaign{}, apperrors.NewNotFound("SMS campaign not found")
	}
	if err != nil {
		return Campaign{}, apperrors.NewInternal("Unable to get SMS campaign", err)
	}
	if current.Status != StatusDraft {
		return Campaign{}, apperrors.NewConflict("Only draft SMS campaigns can be updated")
	}
	if req.Name != nil {
		current.Name = strings.TrimSpace(*req.Name)
	}
	if req.Body != nil {
		current.Body = *req.Body
	}
	if req.RateLimitPerSecond != nil {
		current.RateLimitPerSecond = *req.RateLimitPerSecond
	}
	if req.DailySendLimit != nil {
		if *req.DailySendLimit == 0 {
			current.DailySendLimit = nil
		} else {
			limit := *req.DailySendLimit
			current.DailySendLimit = &limit
		}
	}
	segmentID := uuid.MustParse(current.SegmentID)
	if req.SegmentID != nil {
		segmentID, err = parseID(*req.SegmentID, "Segment id")
		if err != nil {
			return Campaign{}, err
		}
	}
	senderID := uuid.MustParse(current.SenderID)
	if req.SenderID != nil {
		senderID, err = parseID(*req.SenderID, "Sender id")
		if err != nil {
			return Campaign{}, err
		}
	}
	_, _, validated, err := validateCreate(CreateRequest{Name: current.Name, SegmentID: segmentID.String(), SenderID: senderID.String(), Body: current.Body, RateLimitPerSecond: current.RateLimitPerSecond, DailySendLimit: current.DailySendLimit})
	if err != nil {
		return Campaign{}, err
	}
	value, err := s.repository.Update(ctx, tc.Scope.TeamID, campaignID, segmentID, senderID, req, Campaign{Name: validated.Name, Body: validated.Body, RateLimitPerSecond: validated.RateLimitPerSecond, DailySendLimit: validated.DailySendLimit})
	if errors.Is(err, ErrInvalidReference) {
		return Campaign{}, apperrors.NewBadRequest("Segment or sender ID was not found for this team")
	}
	if errors.Is(err, ErrConflict) {
		return Campaign{}, apperrors.NewConflict("SMS campaign was modified or is no longer a draft")
	}
	if err != nil {
		return Campaign{}, apperrors.NewInternal("Unable to update SMS campaign", err)
	}
	return value, nil
}

func (s *Service) Delete(ctx context.Context, id string) (Campaign, error) {
	tc, err := require(ctx, authz.PermissionSMSSend)
	if err != nil {
		return Campaign{}, err
	}
	campaignID, err := parseID(id, "Campaign id")
	if err != nil {
		return Campaign{}, err
	}
	value, err := s.repository.Delete(ctx, tc.Scope.TeamID, campaignID)
	if errors.Is(err, ErrConflict) {
		return Campaign{}, apperrors.NewConflict("Only draft or canceled SMS campaigns can be deleted")
	}
	if err != nil {
		return Campaign{}, apperrors.NewInternal("Unable to delete SMS campaign", err)
	}
	return value, nil
}

func (s *Service) Duplicate(ctx context.Context, id string, req DuplicateRequest) (Campaign, error) {
	tc, err := require(ctx, authz.PermissionSMSSend)
	if err != nil {
		return Campaign{}, err
	}
	campaignID, err := parseID(id, "Campaign id")
	if err != nil {
		return Campaign{}, err
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return Campaign{}, apperrors.NewBadRequest("Campaign name is required")
	}
	value, err := s.repository.Duplicate(ctx, tc.Scope.TeamID, campaignID, req.Name)
	if errors.Is(err, ErrConflict) {
		return Campaign{}, apperrors.NewNotFound("SMS campaign not found")
	}
	if err != nil {
		return Campaign{}, apperrors.NewInternal("Unable to duplicate SMS campaign", err)
	}
	return value, nil
}

func (s *Service) Send(ctx context.Context, id string, req SendRequest) (Campaign, error) {
	tc, err := require(ctx, authz.PermissionSMSSend)
	if err != nil {
		return Campaign{}, err
	}
	campaignID, err := parseID(id, "Campaign id")
	if err != nil {
		return Campaign{}, err
	}
	if req.ScheduledAt != nil {
		when := req.ScheduledAt.UTC()
		if time.Until(when) < 30*time.Second {
			return Campaign{}, apperrors.NewBadRequest("scheduled_at must be at least 30 seconds in the future")
		}
		req.ScheduledAt = &when
	}
	current, err := s.repository.Get(ctx, tc.Scope.TeamID, campaignID)
	if errors.Is(err, ErrNotFound) {
		return Campaign{}, apperrors.NewNotFound("SMS campaign not found")
	}
	if err != nil {
		return Campaign{}, apperrors.NewInternal("Unable to get SMS campaign", err)
	}
	if current.Status != StatusDraft {
		return Campaign{}, apperrors.NewConflict("Only draft SMS campaigns can be sent")
	}
	approved, err := s.repository.HasApprovedSender(ctx, tc.Scope.TeamID, campaignID)
	if err != nil {
		return Campaign{}, apperrors.NewInternal("Unable to validate SMS campaign sender", err)
	}
	if !approved {
		return Campaign{}, apperrors.NewConflict("SMS campaign sender ID must be approved before sending")
	}
	value, err := s.repository.Activate(ctx, tc.Scope.TeamID, campaignID, req.ScheduledAt)
	if errors.Is(err, ErrConflict) {
		return Campaign{}, apperrors.NewConflict("Only draft SMS campaigns can be sent")
	}
	if err != nil {
		return Campaign{}, apperrors.NewInternal("Unable to send SMS campaign", err)
	}
	return value, nil
}

func (s *Service) Schedule(ctx context.Context, id string, req ScheduleRequest) (Campaign, error) {
	if req.ScheduledAt.IsZero() {
		return Campaign{}, apperrors.NewBadRequest("scheduled_at is required")
	}
	return s.Send(ctx, id, SendRequest{ScheduledAt: &req.ScheduledAt})
}

func (s *Service) Cancel(ctx context.Context, id string) (Campaign, error) {
	tc, err := require(ctx, authz.PermissionSMSSend)
	if err != nil {
		return Campaign{}, err
	}
	campaignID, err := parseID(id, "Campaign id")
	if err != nil {
		return Campaign{}, err
	}
	value, err := s.repository.Cancel(ctx, tc.Scope.TeamID, campaignID)
	if errors.Is(err, ErrConflict) {
		return Campaign{}, apperrors.NewConflict("Only scheduled or queued SMS campaigns can be canceled")
	}
	if err != nil {
		return Campaign{}, apperrors.NewInternal("Unable to cancel SMS campaign", err)
	}
	return value, nil
}

func (s *Service) Preview(ctx context.Context, id string) (Preview, error) {
	value, err := s.Get(ctx, id)
	if err != nil {
		return Preview{}, err
	}
	encoding, characters, segments := sms.AnalyzeBody(value.Body)
	return Preview{Body: value.Body, Encoding: encoding, Characters: characters, Segments: segments}, nil
}

func (s *Service) ListRecipients(ctx context.Context, id string, req ListRequest) ([]Recipient, error) {
	tc, err := require(ctx, authz.PermissionSMSRead)
	if err != nil {
		return nil, err
	}
	campaignID, err := parseID(id, "Campaign id")
	if err != nil {
		return nil, err
	}
	if _, err = s.repository.Get(ctx, tc.Scope.TeamID, campaignID); errors.Is(err, ErrNotFound) {
		return nil, apperrors.NewNotFound("SMS campaign not found")
	} else if err != nil {
		return nil, apperrors.NewInternal("Unable to get SMS campaign", err)
	}
	normalizeList(&req)
	recipients, err := s.repository.ListRecipients(ctx, tc.Scope.TeamID, campaignID, req)
	if err != nil {
		return nil, apperrors.NewInternal("Unable to list SMS campaign recipients", err)
	}
	if err = s.applyRecipientDeliveries(ctx, tc.Scope.TeamID, recipients); err != nil {
		return nil, apperrors.NewInternal("Unable to list SMS campaign recipient delivery states", err)
	}
	return recipients, nil
}

func (s *Service) GetCostEstimate(ctx context.Context, id string) (CostEstimate, error) {
	tc, err := require(ctx, authz.PermissionSMSRead)
	if err != nil {
		return CostEstimate{}, err
	}
	campaignID, err := parseID(id, "Campaign id")
	if err != nil {
		return CostEstimate{}, err
	}
	value, err := s.repository.GetCostEstimate(ctx, tc.Scope.TeamID, campaignID)
	if errors.Is(err, ErrNotFound) {
		return CostEstimate{}, apperrors.NewNotFound("SMS campaign not found")
	}
	if err != nil {
		return CostEstimate{}, apperrors.NewInternal("Unable to get SMS campaign cost estimate", err)
	}
	return value, nil
}

func validateCreate(req CreateRequest) (uuid.UUID, uuid.UUID, CreateRequest, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return uuid.Nil, uuid.Nil, CreateRequest{}, apperrors.NewBadRequest("Campaign name is required")
	}
	if strings.TrimSpace(req.Body) == "" {
		return uuid.Nil, uuid.Nil, CreateRequest{}, apperrors.NewBadRequest("SMS body is required")
	}
	if utf8.RuneCountInString(req.Body) > maxBodyCharacters {
		return uuid.Nil, uuid.Nil, CreateRequest{}, apperrors.NewBadRequest("SMS body must be at most 1600 characters")
	}
	if req.RateLimitPerSecond == 0 {
		req.RateLimitPerSecond = 10
	}
	if req.RateLimitPerSecond < 1 || req.RateLimitPerSecond > 1000 {
		return uuid.Nil, uuid.Nil, CreateRequest{}, apperrors.NewBadRequest("rate_limit_per_second must be between 1 and 1000")
	}
	if req.DailySendLimit != nil && *req.DailySendLimit <= 0 {
		return uuid.Nil, uuid.Nil, CreateRequest{}, apperrors.NewBadRequest("daily_send_limit must be greater than zero")
	}
	segmentID, err := parseID(req.SegmentID, "Segment id")
	if err != nil {
		return uuid.Nil, uuid.Nil, CreateRequest{}, err
	}
	senderID, err := parseID(req.SenderID, "Sender id")
	if err != nil {
		return uuid.Nil, uuid.Nil, CreateRequest{}, err
	}
	return segmentID, senderID, req, nil
}

func (s *Service) RecordOptOut(ctx context.Context, req RecordOptOutRequest) (ConsentEvent, error) {
	tc, err := require(ctx, authz.PermissionSMSSend)
	if err != nil {
		return ConsentEvent{}, err
	}
	number, parseErr := phonenumbers.Parse(strings.TrimSpace(req.Phone), "")
	if parseErr != nil || !phonenumbers.IsValidNumber(number) {
		return ConsentEvent{}, apperrors.NewBadRequest("Phone must be a valid international number")
	}
	req.Phone = phonenumbers.Format(number, phonenumbers.E164)
	req.Source = strings.ToLower(strings.TrimSpace(req.Source))
	if req.Source == "" {
		req.Source = "api"
	}
	if req.Source != "api" && req.Source != "manual" && req.Source != "import" {
		return ConsentEvent{}, apperrors.NewBadRequest("source must be api, manual, or import")
	}
	req.SourceID = normalizeString(req.SourceID)
	value, err := s.repository.RecordOptOut(ctx, tc.Scope.TeamID, req)
	if err != nil {
		return ConsentEvent{}, apperrors.NewInternal("Unable to record SMS opt-out", err)
	}
	return value, nil
}

func normalizeString(value *string) *string {
	if value == nil {
		return nil
	}
	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return nil
	}
	return &normalized
}

func (s *Service) GetExclusionSummary(ctx context.Context, id string) (ExclusionSummary, error) {
	tc, err := require(ctx, authz.PermissionSMSRead)
	if err != nil {
		return ExclusionSummary{}, err
	}
	campaignID, err := parseID(id, "Campaign id")
	if err != nil {
		return ExclusionSummary{}, err
	}
	value, err := s.repository.GetExclusionSummary(ctx, tc.Scope.TeamID, campaignID)
	if errors.Is(err, ErrNotFound) {
		return ExclusionSummary{}, apperrors.NewNotFound("SMS campaign not found")
	}
	if err != nil {
		return ExclusionSummary{}, apperrors.NewInternal("Unable to get SMS campaign exclusions", err)
	}
	return value, nil
}

func (s *Service) GetAnalytics(ctx context.Context, id string) (Analytics, error) {
	tc, err := require(ctx, authz.PermissionSMSRead)
	if err != nil {
		return Analytics{}, err
	}
	campaignID, err := parseID(id, "Campaign id")
	if err != nil {
		return Analytics{}, err
	}
	value, err := s.repository.GetAnalytics(ctx, tc.Scope.TeamID, campaignID)
	if errors.Is(err, ErrNotFound) {
		return Analytics{}, apperrors.NewNotFound("SMS campaign not found")
	}
	if err != nil {
		return Analytics{}, apperrors.NewInternal("Unable to get SMS campaign analytics", err)
	}
	return value, nil
}

func parseID(value, label string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return uuid.Nil, apperrors.NewBadRequest(label + " must be a valid UUID")
	}
	return id, nil
}

func require(ctx context.Context, permission authz.Permission) (authz.Access, error) {
	tc, decision := authz.ResolveAccess(ctx, permission)
	if !decision.Allowed {
		return authz.Access{}, apperrors.NewForbidden(decision.Reason)
	}
	return tc, nil
}

func normalizeList(req *ListRequest) {
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 50
	}
	if req.Offset < 0 {
		req.Offset = 0
	}
}

type recipientDelivery struct {
	Status      string
	DeliveredAt pgtype.Timestamptz
}

func (s *Service) applyRecipientDeliveries(ctx context.Context, teamID uuid.UUID, recipients []Recipient) error {
	messageIDs := make([]uuid.UUID, 0, len(recipients))
	for _, recipient := range recipients {
		if recipient.SMSMessageID == nil {
			continue
		}
		messageID, err := uuid.Parse(*recipient.SMSMessageID)
		if err != nil {
			return fmt.Errorf("parse campaign recipient SMS message id: %w", err)
		}
		messageIDs = append(messageIDs, messageID)
	}
	if len(messageIDs) == 0 {
		return nil
	}

	rows, err := s.repository.db.Query(ctx, `
		SELECT id, status, delivered_at
		FROM sms_messages
		WHERE team_id = $1 AND id = ANY($2::uuid[])
	`, teamID, messageIDs)
	if err != nil {
		return fmt.Errorf("list campaign recipient SMS delivery states: %w", err)
	}
	defer rows.Close()

	deliveries := make(map[string]recipientDelivery, len(messageIDs))
	for rows.Next() {
		var messageID uuid.UUID
		var delivery recipientDelivery
		if err = rows.Scan(&messageID, &delivery.Status, &delivery.DeliveredAt); err != nil {
			return fmt.Errorf("scan campaign recipient SMS delivery state: %w", err)
		}
		deliveries[messageID.String()] = delivery
	}
	if err = rows.Err(); err != nil {
		return fmt.Errorf("iterate campaign recipient SMS delivery states: %w", err)
	}

	for index := range recipients {
		messageID := recipients[index].SMSMessageID
		if messageID == nil {
			continue
		}
		delivery, ok := deliveries[*messageID]
		if !ok {
			continue
		}
		status := delivery.Status
		recipients[index].DeliveryStatus = &status
		recipients[index].DeliveredAt = pgconv.TimestamptzToTimePtr(delivery.DeliveredAt)
	}
	return nil
}
