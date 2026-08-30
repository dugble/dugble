package broadcast

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/dugble/dugble/server/internal/authz"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

type Service struct {
	repository *Repository
}

// NewService accepts ignored compatibility dependencies so existing registry
// wiring can move independently from the broadcast module rebuild.
func NewService(repository *Repository, _ ...any) *Service {
	return &Service{repository: repository}
}

func requireTenant(ctx context.Context, permission authz.Permission) (authz.Access, error) {
	access, decision := authz.ResolveAccess(ctx, permission)
	if !decision.Allowed {
		return authz.Access{}, apperrors.NewForbidden(decision.Reason)
	}
	return access, nil
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Broadcast, error) {
	access, err := requireTenant(ctx, authz.PermissionBroadcastsWrite)
	if err != nil {
		return Broadcast{}, err
	}
	if s == nil || s.repository == nil {
		return Broadcast{}, apperrors.NewInternal("Broadcast service is not configured", nil)
	}

	req.Subject = strings.TrimSpace(req.Subject)
	req.HTML = strings.TrimSpace(req.HTML)
	if req.Subject == "" {
		return Broadcast{}, apperrors.NewBadRequest("Subject is required")
	}
	if req.HTML == "" {
		return Broadcast{}, apperrors.NewBadRequest("HTML is required")
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		req.Name = req.Subject
	}
	normalizeCreateContent(&req)

	segmentID, err := parseID(req.SegmentID, "Segment id")
	if err != nil {
		return Broadcast{}, err
	}
	topicID, err := parseOptionalID(req.TopicID, "Topic id")
	if err != nil {
		return Broadcast{}, err
	}
	if req.VariableBindings == nil {
		req.VariableBindings = map[string]any{}
	}
	if req.ScheduledAt != nil && !req.Send {
		return Broadcast{}, apperrors.NewBadRequest("scheduled_at requires send=true")
	}
	if req.ScheduledAt != nil && !req.ScheduledAt.After(time.Now()) {
		return Broadcast{}, apperrors.NewBadRequest("scheduled_at must be in the future")
	}

	value, err := s.repository.Create(ctx, access.Scope.TeamID, segmentID, topicID, req)
	if err != nil {
		return Broadcast{}, apperrors.NewInternal("Unable to create broadcast", err)
	}
	if !req.Send {
		return value, nil
	}

	value, err = s.repository.Send(ctx, access.Scope.TeamID, uuid.MustParse(value.ID), req.ScheduledAt)
	if errors.Is(err, ErrConflict) {
		return Broadcast{}, apperrors.NewConflict("Broadcast could not be queued")
	}
	if err != nil {
		return Broadcast{}, apperrors.NewInternal("Unable to queue broadcast", err)
	}
	return value, nil
}

func (s *Service) List(ctx context.Context, req ListRequest) ([]Broadcast, error) {
	access, err := requireTenant(ctx, authz.PermissionBroadcastsRead)
	if err != nil {
		return nil, err
	}
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 50
	}
	if req.Offset < 0 {
		req.Offset = 0
	}
	values, err := s.repository.List(ctx, access.Scope.TeamID, req.Limit, req.Offset)
	if err != nil {
		return nil, apperrors.NewInternal("Unable to list broadcasts", err)
	}
	return values, nil
}

func (s *Service) Get(ctx context.Context, identifier string) (Broadcast, error) {
	access, err := requireTenant(ctx, authz.PermissionBroadcastsRead)
	if err != nil {
		return Broadcast{}, err
	}
	id, err := parseID(identifier, "Broadcast id")
	if err != nil {
		return Broadcast{}, err
	}
	value, err := s.repository.Get(ctx, access.Scope.TeamID, id)
	if errors.Is(err, ErrNotFound) {
		return Broadcast{}, apperrors.NewNotFound("Broadcast not found")
	}
	if err != nil {
		return Broadcast{}, apperrors.NewInternal("Unable to get broadcast", err)
	}
	return value, nil
}

func (s *Service) Update(ctx context.Context, identifier string, req UpdateRequest) (Broadcast, error) {
	access, err := requireTenant(ctx, authz.PermissionBroadcastsWrite)
	if err != nil {
		return Broadcast{}, err
	}
	id, err := parseID(identifier, "Broadcast id")
	if err != nil {
		return Broadcast{}, err
	}
	if req.Revision <= 0 {
		return Broadcast{}, apperrors.NewBadRequest("Revision is required")
	}

	current, err := s.repository.Get(ctx, access.Scope.TeamID, id)
	if errors.Is(err, ErrNotFound) {
		return Broadcast{}, apperrors.NewNotFound("Broadcast not found")
	}
	if err != nil {
		return Broadcast{}, apperrors.NewInternal("Unable to get broadcast", err)
	}
	if current.Status != StatusDraft && current.Status != StatusScheduled {
		return Broadcast{}, apperrors.NewConflict("Only draft or scheduled broadcasts can be updated")
	}

	segmentID, topicID, err := applyUpdate(&current, req)
	if err != nil {
		return Broadcast{}, err
	}
	value, err := s.repository.Update(ctx, access.Scope.TeamID, id, segmentID, topicID, req.Revision, current)
	if errors.Is(err, ErrConflict) {
		return Broadcast{}, apperrors.NewConflict("Broadcast was modified or is no longer editable")
	}
	if err != nil {
		return Broadcast{}, apperrors.NewInternal("Unable to update broadcast", err)
	}
	return value, nil
}

func (s *Service) Delete(ctx context.Context, identifier string) (Broadcast, error) {
	access, err := requireTenant(ctx, authz.PermissionBroadcastsWrite)
	if err != nil {
		return Broadcast{}, err
	}
	id, err := parseID(identifier, "Broadcast id")
	if err != nil {
		return Broadcast{}, err
	}
	value, err := s.repository.Delete(ctx, access.Scope.TeamID, id)
	if errors.Is(err, ErrConflict) {
		return Broadcast{}, apperrors.NewConflict("Only draft or canceled broadcasts can be deleted")
	}
	if err != nil {
		return Broadcast{}, apperrors.NewInternal("Unable to delete broadcast", err)
	}
	return value, nil
}

func (s *Service) Send(ctx context.Context, identifier string, req SendRequest) (Broadcast, error) {
	access, err := requireTenant(ctx, authz.PermissionBroadcastsSend)
	if err != nil {
		return Broadcast{}, err
	}
	id, err := parseID(identifier, "Broadcast id")
	if err != nil {
		return Broadcast{}, err
	}
	if req.ScheduledAt != nil && !req.ScheduledAt.After(time.Now()) {
		return Broadcast{}, apperrors.NewBadRequest("scheduled_at must be in the future")
	}
	value, err := s.repository.Send(ctx, access.Scope.TeamID, id, req.ScheduledAt)
	if errors.Is(err, ErrNotFound) {
		return Broadcast{}, apperrors.NewNotFound("Broadcast not found")
	}
	if errors.Is(err, ErrConflict) {
		return Broadcast{}, apperrors.NewConflict("Only draft broadcasts can be sent; scheduled broadcasts can only be rescheduled")
	}
	if err != nil {
		return Broadcast{}, apperrors.NewInternal("Unable to send broadcast", err)
	}
	return value, nil
}

func (s *Service) Cancel(ctx context.Context, identifier string) (Broadcast, error) {
	access, err := requireTenant(ctx, authz.PermissionBroadcastsSend)
	if err != nil {
		return Broadcast{}, err
	}
	id, err := parseID(identifier, "Broadcast id")
	if err != nil {
		return Broadcast{}, err
	}
	value, err := s.repository.Cancel(ctx, access.Scope.TeamID, id)
	if errors.Is(err, ErrNotFound) {
		return Broadcast{}, apperrors.NewNotFound("Broadcast not found")
	}
	if errors.Is(err, ErrConflict) {
		return Broadcast{}, apperrors.NewConflict("Only scheduled or queued broadcasts can be canceled")
	}
	if err != nil {
		return Broadcast{}, apperrors.NewInternal("Unable to cancel broadcast", err)
	}
	return value, nil
}

func (s *Service) Preview(ctx context.Context, identifier string, req PreviewRequest) (PreviewResponse, error) {
	value, err := s.Get(ctx, identifier)
	if err != nil {
		return PreviewResponse{}, err
	}
	preview, err := RenderBroadcast(value, req.Variables)
	if err != nil {
		return PreviewResponse{}, apperrors.NewBadRequest(err.Error())
	}
	return preview, nil
}

func (s *Service) ListRecipients(ctx context.Context, identifier string, req ListRequest) ([]Recipient, error) {
	access, err := requireTenant(ctx, authz.PermissionBroadcastsRead)
	if err != nil {
		return nil, err
	}
	id, err := parseID(identifier, "Broadcast id")
	if err != nil {
		return nil, err
	}
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 50
	}
	if req.Offset < 0 {
		req.Offset = 0
	}
	if _, err = s.repository.Get(ctx, access.Scope.TeamID, id); errors.Is(err, ErrNotFound) {
		return nil, apperrors.NewNotFound("Broadcast not found")
	} else if err != nil {
		return nil, apperrors.NewInternal("Unable to get broadcast", err)
	}
	values, err := s.repository.ListRecipients(ctx, access.Scope.TeamID, id, req.Limit, req.Offset)
	if err != nil {
		return nil, apperrors.NewInternal("Unable to list broadcast recipients", err)
	}
	return values, nil
}

func (s *Service) GetExclusionSummary(ctx context.Context, identifier string) (ExclusionSummary, error) {
	access, err := requireTenant(ctx, authz.PermissionBroadcastsRead)
	if err != nil {
		return ExclusionSummary{}, err
	}
	id, err := parseID(identifier, "Broadcast id")
	if err != nil {
		return ExclusionSummary{}, err
	}
	if _, err = s.repository.Get(ctx, access.Scope.TeamID, id); errors.Is(err, ErrNotFound) {
		return ExclusionSummary{}, apperrors.NewNotFound("Broadcast not found")
	} else if err != nil {
		return ExclusionSummary{}, apperrors.NewInternal("Unable to get broadcast", err)
	}
	summary, err := s.repository.GetExclusionSummary(ctx, access.Scope.TeamID, id)
	if err != nil {
		return ExclusionSummary{}, apperrors.NewInternal("Unable to summarize excluded recipients", err)
	}
	return summary, nil
}

func (s *Service) GetAnalytics(ctx context.Context, identifier string) (Analytics, error) {
	access, err := requireTenant(ctx, authz.PermissionBroadcastsRead)
	if err != nil {
		return Analytics{}, err
	}
	id, err := parseID(identifier, "Broadcast id")
	if err != nil {
		return Analytics{}, err
	}
	analytics, err := s.repository.GetAnalytics(ctx, access.Scope.TeamID, id)
	if errors.Is(err, ErrNotFound) {
		return Analytics{}, apperrors.NewNotFound("Broadcast not found")
	}
	if err != nil {
		return Analytics{}, apperrors.NewInternal("Unable to get broadcast analytics", err)
	}
	return analytics, nil
}

func (s *Service) Duplicate(ctx context.Context, identifier string, req DuplicateRequest) (Broadcast, error) {
	access, err := requireTenant(ctx, authz.PermissionBroadcastsWrite)
	if err != nil {
		return Broadcast{}, err
	}
	id, err := parseID(identifier, "Broadcast id")
	if err != nil {
		return Broadcast{}, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		source, getErr := s.repository.Get(ctx, access.Scope.TeamID, id)
		if errors.Is(getErr, ErrNotFound) {
			return Broadcast{}, apperrors.NewNotFound("Broadcast not found")
		}
		if getErr != nil {
			return Broadcast{}, apperrors.NewInternal("Unable to get broadcast", getErr)
		}
		name = source.Name + " Copy"
	}
	value, err := s.repository.Duplicate(ctx, access.Scope.TeamID, id, name)
	if errors.Is(err, ErrNotFound) {
		return Broadcast{}, apperrors.NewNotFound("Broadcast not found")
	}
	if err != nil {
		return Broadcast{}, apperrors.NewInternal("Unable to duplicate broadcast", err)
	}
	return value, nil
}

func normalizeCreateContent(req *CreateRequest) {
	req.FromEmail = normalizeOptionalString(req.FromEmail)
	req.FromName = normalizeOptionalString(req.FromName)
	req.ReplyToEmail = normalizeOptionalString(req.ReplyToEmail)
	req.PreviewText = normalizeOptionalString(req.PreviewText)
	req.Text = normalizeOptionalString(req.Text)
}

func applyUpdate(current *Broadcast, req UpdateRequest) (uuid.UUID, *uuid.UUID, error) {
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return uuid.Nil, nil, apperrors.NewBadRequest("Broadcast name cannot be empty")
		}
		current.Name = name
	}
	segmentID := uuid.MustParse(current.SegmentID)
	var err error
	if req.SegmentID != nil {
		segmentID, err = parseID(*req.SegmentID, "Segment id")
		if err != nil {
			return uuid.Nil, nil, err
		}
		current.SegmentID = segmentID.String()
	}
	topicID, err := parseOptionalID(current.TopicID, "Topic id")
	if err != nil {
		return uuid.Nil, nil, err
	}
	if req.TopicID != nil {
		topicID, err = parseOptionalID(*req.TopicID, "Topic id")
		if err != nil {
			return uuid.Nil, nil, err
		}
		current.TopicID = uuidStringPointer(topicID)
	}
	if req.FromEmail != nil {
		current.FromEmail = optionalStringValue(normalizeOptionalString(*req.FromEmail))
	}
	if req.FromName != nil {
		current.FromName = normalizeOptionalString(*req.FromName)
	}
	if req.ReplyToEmail != nil {
		current.ReplyToEmail = normalizeOptionalString(*req.ReplyToEmail)
	}
	if req.Subject != nil {
		current.Subject = strings.TrimSpace(*req.Subject)
		if current.Subject == "" {
			return uuid.Nil, nil, apperrors.NewBadRequest("Subject cannot be empty")
		}
	}
	if req.PreviewText != nil {
		current.PreviewText = normalizeOptionalString(*req.PreviewText)
	}
	if req.HTML != nil {
		current.HTML = strings.TrimSpace(*req.HTML)
		if current.HTML == "" {
			return uuid.Nil, nil, apperrors.NewBadRequest("HTML cannot be empty")
		}
	}
	if req.Text != nil {
		current.Text = normalizeOptionalString(*req.Text)
	}
	if req.VariableBindings != nil {
		current.VariableBindings = *req.VariableBindings
		if current.VariableBindings == nil {
			current.VariableBindings = map[string]any{}
		}
	}
	return segmentID, topicID, nil
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func uuidStringPointer(value *uuid.UUID) *string {
	if value == nil {
		return nil
	}
	text := value.String()
	return &text
}
