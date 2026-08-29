package broadcast

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/dugble/dugble/server/internal/authz"
	messagetemplate "github.com/dugble/dugble/server/internal/modules/messagetemplate"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

type broadcastContentTemplateService interface {
	CreateBroadcastContent(context.Context, uuid.UUID, messagetemplate.BroadcastContentRequest) (messagetemplate.Template, error)
	BroadcastPublishedVersion(context.Context, uuid.UUID, string) (string, bool, error)
}

type broadcastContentCleanupService interface {
	DeleteBroadcastContentIfUnreferenced(context.Context, uuid.UUID, string) error
}

func (s *Service) CreateAPI(ctx context.Context, req CreateRequest) (Broadcast, error) {
	if !createHasInlineContent(req) {
		return s.Create(ctx, req)
	}
	if strings.TrimSpace(req.Template) != "" {
		return Broadcast{}, apperrors.NewBadRequest("template cannot be combined with inline broadcast content")
	}
	if strings.TrimSpace(req.Subject) == "" {
		return Broadcast{}, apperrors.NewBadRequest("subject is required for inline broadcast content")
	}
	if strings.TrimSpace(req.HTML) == "" {
		return Broadcast{}, apperrors.NewBadRequest("html is required for inline broadcast content")
	}

	tc, err := requireTenant(ctx, authz.PermissionBroadcastsWrite)
	if err != nil {
		return Broadcast{}, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return Broadcast{}, apperrors.NewBadRequest("Broadcast name is required")
	}
	segmentID, err := parseID(req.SegmentID, "Segment id")
	if err != nil {
		return Broadcast{}, err
	}
	topicID, err := parseOptionalID(req.TopicID, "Topic id")
	if err != nil {
		return Broadcast{}, err
	}
	templates, ok := s.templates.(broadcastContentTemplateService)
	if !ok {
		return Broadcast{}, apperrors.NewInternal("Template service does not support broadcast content", nil)
	}
	template, err := templates.CreateBroadcastContent(ctx, tc.Scope.TeamID, messagetemplate.BroadcastContentRequest{
		Name: name, Subject: req.Subject, HTML: req.HTML, Text: req.Text,
		FromEmail: req.FromEmail, FromName: req.FromName, PreviewText: req.PreviewText,
	})
	if err != nil {
		return Broadcast{}, err
	}
	if req.VariableBindings == nil {
		req.VariableBindings = map[string]any{}
	}
	req.Name = name
	value, err := s.repository.Create(ctx, tc.Scope.TeamID, segmentID, topicID, uuid.MustParse(template.ID), req)
	if err != nil {
		s.cleanupBroadcastContent(ctx, tc.Scope.TeamID, template.ID)
		return Broadcast{}, apperrors.NewInternal("Unable to create broadcast", err)
	}
	return value, nil
}

func (s *Service) UpdateAPI(ctx context.Context, identifier string, req UpdateRequest) (Broadcast, error) {
	if !updateHasInlineContent(req) {
		return s.Update(ctx, identifier, req)
	}
	if req.Template != nil && strings.TrimSpace(*req.Template) != "" {
		return Broadcast{}, apperrors.NewBadRequest("template cannot be combined with inline broadcast content")
	}
	if req.Subject == nil || strings.TrimSpace(*req.Subject) == "" {
		return Broadcast{}, apperrors.NewBadRequest("subject is required when updating inline broadcast content")
	}
	if req.HTML == nil || strings.TrimSpace(*req.HTML) == "" {
		return Broadcast{}, apperrors.NewBadRequest("html is required when updating inline broadcast content")
	}

	tc, err := requireTenant(ctx, authz.PermissionBroadcastsWrite)
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
	current, err := s.repository.Get(ctx, tc.Scope.TeamID, id)
	if errors.Is(err, ErrNotFound) {
		return Broadcast{}, apperrors.NewNotFound("Broadcast not found")
	}
	if err != nil {
		return Broadcast{}, apperrors.NewInternal("Unable to get broadcast", err)
	}
	if current.Status != StatusDraft {
		return Broadcast{}, apperrors.NewConflict("Only draft broadcasts can be updated")
	}

	if req.Name != nil {
		current.Name = strings.TrimSpace(*req.Name)
		if current.Name == "" {
			return Broadcast{}, apperrors.NewBadRequest("Broadcast name is required")
		}
	}
	segmentID := uuid.MustParse(current.SegmentID)
	if req.SegmentID != nil {
		segmentID, err = parseID(*req.SegmentID, "Segment id")
		if err != nil {
			return Broadcast{}, err
		}
	}
	topicID, err := parseOptionalID(current.TopicID, "Topic id")
	if err != nil {
		return Broadcast{}, err
	}
	if req.TopicID != nil {
		topicID, err = parseOptionalID(*req.TopicID, "Topic id")
		if err != nil {
			return Broadcast{}, err
		}
	}

	templates, ok := s.templates.(broadcastContentTemplateService)
	if !ok {
		return Broadcast{}, apperrors.NewInternal("Template service does not support broadcast content", nil)
	}
	previousTemplateID := current.TemplateID
	template, err := templates.CreateBroadcastContent(ctx, tc.Scope.TeamID, messagetemplate.BroadcastContentRequest{
		Name: current.Name, Subject: *req.Subject, HTML: *req.HTML,
		Text: pointerPointerValue(req.Text), FromEmail: pointerPointerValue(req.FromEmail),
		FromName: pointerPointerValue(req.FromName), PreviewText: pointerPointerValue(req.PreviewText),
	})
	if err != nil {
		return Broadcast{}, err
	}
	if req.VariableBindings != nil {
		current.VariableBindings = *req.VariableBindings
	}
	value, err := s.repository.Update(ctx, tc.Scope.TeamID, id, segmentID, topicID, uuid.MustParse(template.ID), req, current)
	if err != nil {
		s.cleanupBroadcastContent(ctx, tc.Scope.TeamID, template.ID)
		if errors.Is(err, ErrConflict) {
			return Broadcast{}, apperrors.NewConflict("Broadcast was modified or is no longer a draft")
		}
		return Broadcast{}, apperrors.NewInternal("Unable to update broadcast", err)
	}

	// A successful inline edit replaces the prior immutable content snapshot.
	// Delete it only if it is internal and no other broadcast (for example a
	// duplicate) still references it. Cleanup failure must not turn a committed
	// broadcast update into an API failure.
	s.cleanupBroadcastContent(ctx, tc.Scope.TeamID, previousTemplateID)
	return value, nil
}

func (s *Service) cleanupBroadcastContent(ctx context.Context, teamID uuid.UUID, templateID string) {
	cleanup, ok := s.templates.(broadcastContentCleanupService)
	if !ok {
		return
	}
	_ = cleanup.DeleteBroadcastContentIfUnreferenced(ctx, teamID, templateID)
}

func (s *Service) SendAPI(ctx context.Context, identifier string, req SendRequest) (Broadcast, error) {
	tc, err := requireTenant(ctx, authz.PermissionBroadcastsSend)
	if err != nil {
		return Broadcast{}, err
	}
	id, err := parseID(identifier, "Broadcast id")
	if err != nil {
		return Broadcast{}, err
	}
	current, err := s.repository.Get(ctx, tc.Scope.TeamID, id)
	if errors.Is(err, ErrNotFound) {
		return Broadcast{}, apperrors.NewNotFound("Broadcast not found")
	}
	if err != nil {
		return Broadcast{}, apperrors.NewInternal("Unable to get broadcast", err)
	}

	templates, ok := s.templates.(broadcastContentTemplateService)
	if !ok {
		return s.Send(ctx, identifier, req)
	}
	versionID, internal, err := templates.BroadcastPublishedVersion(ctx, tc.Scope.TeamID, current.TemplateID)
	if err != nil {
		return Broadcast{}, err
	}
	if !internal {
		return s.Send(ctx, identifier, req)
	}
	if req.ScheduledAt != nil && !req.ScheduledAt.After(time.Now()) {
		return Broadcast{}, apperrors.NewBadRequest("scheduled_at must be in the future")
	}
	value, err := s.repository.Send(ctx, tc.Scope.TeamID, id, uuid.MustParse(versionID), req.ScheduledAt)
	if errors.Is(err, ErrConflict) {
		return Broadcast{}, apperrors.NewConflict("Only draft broadcasts can be sent")
	}
	if err != nil {
		return Broadcast{}, apperrors.NewInternal("Unable to send broadcast", err)
	}
	return value, nil
}

func createHasInlineContent(req CreateRequest) bool {
	return strings.TrimSpace(req.Subject) != "" || strings.TrimSpace(req.HTML) != "" || req.Text != nil || req.FromEmail != nil || req.FromName != nil || req.PreviewText != nil
}

func updateHasInlineContent(req UpdateRequest) bool {
	return req.Subject != nil || req.HTML != nil || req.Text != nil || req.FromEmail != nil || req.FromName != nil || req.PreviewText != nil
}

func pointerPointerValue(value **string) *string {
	if value == nil {
		return nil
	}
	return *value
}
