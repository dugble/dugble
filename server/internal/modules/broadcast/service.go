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

type TemplateService interface {
	Get(context.Context, string) (messagetemplate.Template, error)
	Preview(context.Context, string, messagetemplate.PreviewRequest) (messagetemplate.PreviewResponse, error)
}

type Service struct {
	repository *Repository
	templates  TemplateService
}

func NewService(repository *Repository, templates TemplateService) *Service {
	return &Service{repository: repository, templates: templates}
}

func requireTenant(ctx context.Context, permission authz.Permission) (authz.Access, error) {
	tc, decision := authz.ResolveAccess(ctx, permission)
	if !decision.Allowed {
		return authz.Access{}, apperrors.NewForbidden(decision.Reason)
	}
	return tc, nil
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Broadcast, error) {
	inline := createHasInlineContent(req)
	if inline {
		if strings.TrimSpace(req.Template) != "" {
			return Broadcast{}, apperrors.NewBadRequest("template cannot be combined with inline broadcast content")
		}
		if strings.TrimSpace(req.Subject) == "" {
			return Broadcast{}, apperrors.NewBadRequest("subject is required for inline broadcast content")
		}
		if strings.TrimSpace(req.HTML) == "" {
			return Broadcast{}, apperrors.NewBadRequest("html is required for inline broadcast content")
		}
	}

	tc, err := requireTenant(ctx, authz.PermissionBroadcastsWrite)
	if err != nil {
		return Broadcast{}, err
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
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

	template, err := s.resolveCreateTemplate(ctx, tc.Scope.TeamID, req, inline)
	if err != nil {
		return Broadcast{}, err
	}
	if req.VariableBindings == nil {
		req.VariableBindings = map[string]any{}
	}
	value, err := s.repository.Create(ctx, tc.Scope.TeamID, segmentID, topicID, uuid.MustParse(template.ID), req)
	if err != nil {
		if inline {
			s.cleanupBroadcastContent(ctx, tc.Scope.TeamID, template.ID)
		}
		return Broadcast{}, apperrors.NewInternal("Unable to create broadcast", err)
	}
	return value, nil
}

func (s *Service) resolveCreateTemplate(ctx context.Context, teamID uuid.UUID, req CreateRequest, inline bool) (messagetemplate.Template, error) {
	if inline {
		templates, ok := s.templates.(broadcastContentTemplateService)
		if !ok {
			return messagetemplate.Template{}, apperrors.NewInternal("Template service does not support broadcast content", nil)
		}
		return templates.CreateBroadcastContent(ctx, teamID, messagetemplate.BroadcastContentRequest{
			Name: req.Name, Subject: req.Subject, HTML: req.HTML, Text: req.Text,
			FromEmail: req.FromEmail, FromName: req.FromName, PreviewText: req.PreviewText,
		})
	}
	if s.templates == nil {
		return messagetemplate.Template{}, apperrors.NewInternal("Template service is not configured", nil)
	}
	return s.templates.Get(ctx, strings.TrimSpace(req.Template))
}

func (s *Service) List(ctx context.Context, req ListRequest) ([]Broadcast, error) {
	tc, err := requireTenant(ctx, authz.PermissionBroadcastsRead)
	if err != nil {
		return nil, err
	}
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 50
	}
	if req.Offset < 0 {
		req.Offset = 0
	}
	values, err := s.repository.List(ctx, tc.Scope.TeamID, req.Limit, req.Offset)
	if err != nil {
		return nil, apperrors.NewInternal("Unable to list broadcasts", err)
	}
	return values, nil
}

func (s *Service) Get(ctx context.Context, identifier string) (Broadcast, error) {
	tc, err := requireTenant(ctx, authz.PermissionBroadcastsRead)
	if err != nil {
		return Broadcast{}, err
	}
	id, err := parseID(identifier, "Broadcast id")
	if err != nil {
		return Broadcast{}, err
	}
	value, err := s.repository.Get(ctx, tc.Scope.TeamID, id)
	if errors.Is(err, ErrNotFound) {
		return Broadcast{}, apperrors.NewNotFound("Broadcast not found")
	}
	if err != nil {
		return Broadcast{}, apperrors.NewInternal("Unable to get broadcast", err)
	}
	return value, nil
}

func (s *Service) Update(ctx context.Context, identifier string, req UpdateRequest) (Broadcast, error) {
	inline := updateHasInlineContent(req)
	if inline {
		if req.Template != nil && strings.TrimSpace(*req.Template) != "" {
			return Broadcast{}, apperrors.NewBadRequest("template cannot be combined with inline broadcast content")
		}
		if req.Subject == nil || strings.TrimSpace(*req.Subject) == "" {
			return Broadcast{}, apperrors.NewBadRequest("subject is required when updating inline broadcast content")
		}
		if req.HTML == nil || strings.TrimSpace(*req.HTML) == "" {
			return Broadcast{}, apperrors.NewBadRequest("html is required when updating inline broadcast content")
		}
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

	previousTemplateID := current.TemplateID
	templateID, createdInline, err := s.resolveUpdateTemplate(ctx, tc.Scope.TeamID, current, req, inline)
	if err != nil {
		return Broadcast{}, err
	}
	if req.VariableBindings != nil {
		current.VariableBindings = *req.VariableBindings
	}
	value, err := s.repository.Update(ctx, tc.Scope.TeamID, id, segmentID, topicID, templateID, req, current)
	if err != nil {
		if createdInline {
			s.cleanupBroadcastContent(ctx, tc.Scope.TeamID, templateID.String())
		}
		if errors.Is(err, ErrConflict) {
			return Broadcast{}, apperrors.NewConflict("Broadcast was modified or is no longer a draft")
		}
		return Broadcast{}, apperrors.NewInternal("Unable to update broadcast", err)
	}
	if templateID.String() != previousTemplateID {
		s.cleanupBroadcastContent(ctx, tc.Scope.TeamID, previousTemplateID)
	}
	return value, nil
}

func (s *Service) resolveUpdateTemplate(ctx context.Context, teamID uuid.UUID, current Broadcast, req UpdateRequest, inline bool) (uuid.UUID, bool, error) {
	if inline {
		templates, ok := s.templates.(broadcastContentTemplateService)
		if !ok {
			return uuid.Nil, false, apperrors.NewInternal("Template service does not support broadcast content", nil)
		}
		template, err := templates.CreateBroadcastContent(ctx, teamID, messagetemplate.BroadcastContentRequest{
			Name: current.Name, Subject: *req.Subject, HTML: *req.HTML,
			Text: pointerPointerValue(req.Text), FromEmail: pointerPointerValue(req.FromEmail),
			FromName: pointerPointerValue(req.FromName), PreviewText: pointerPointerValue(req.PreviewText),
		})
		if err != nil {
			return uuid.Nil, false, err
		}
		return uuid.MustParse(template.ID), true, nil
	}
	if req.Template == nil {
		return uuid.MustParse(current.TemplateID), false, nil
	}
	if s.templates == nil {
		return uuid.Nil, false, apperrors.NewInternal("Template service is not configured", nil)
	}
	template, err := s.templates.Get(ctx, strings.TrimSpace(*req.Template))
	if err != nil {
		return uuid.Nil, false, err
	}
	return uuid.MustParse(template.ID), false, nil
}

func (s *Service) Delete(ctx context.Context, identifier string) (Broadcast, error) {
	tc, err := requireTenant(ctx, authz.PermissionBroadcastsWrite)
	if err != nil {
		return Broadcast{}, err
	}
	id, err := parseID(identifier, "Broadcast id")
	if err != nil {
		return Broadcast{}, err
	}
	value, err := s.repository.Delete(ctx, tc.Scope.TeamID, id)
	if errors.Is(err, ErrConflict) {
		return Broadcast{}, apperrors.NewConflict("Only draft or canceled broadcasts can be deleted")
	}
	if err != nil {
		return Broadcast{}, apperrors.NewInternal("Unable to delete broadcast", err)
	}
	return value, nil
}

func (s *Service) Send(ctx context.Context, identifier string, req SendRequest) (Broadcast, error) {
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
	versionID, err := s.publishedVersionID(ctx, tc.Scope.TeamID, current.TemplateID)
	if err != nil {
		return Broadcast{}, err
	}
	if req.ScheduledAt != nil && !req.ScheduledAt.After(time.Now()) {
		return Broadcast{}, apperrors.NewBadRequest("scheduled_at must be in the future")
	}
	value, err := s.repository.Send(ctx, tc.Scope.TeamID, id, versionID, req.ScheduledAt)
	if errors.Is(err, ErrConflict) {
		return Broadcast{}, apperrors.NewConflict("Only draft broadcasts can be sent")
	}
	if err != nil {
		return Broadcast{}, apperrors.NewInternal("Unable to send broadcast", err)
	}
	return value, nil
}

func (s *Service) publishedVersionID(ctx context.Context, teamID uuid.UUID, templateID string) (uuid.UUID, error) {
	if templates, ok := s.templates.(broadcastContentTemplateService); ok {
		versionID, internal, err := templates.BroadcastPublishedVersion(ctx, teamID, templateID)
		if err != nil {
			return uuid.Nil, err
		}
		if internal {
			return uuid.MustParse(versionID), nil
		}
	}
	if s.templates == nil {
		return uuid.Nil, apperrors.NewInternal("Template service is not configured", nil)
	}
	template, err := s.templates.Get(ctx, templateID)
	if err != nil {
		return uuid.Nil, err
	}
	if template.PublishedVersionID == nil {
		return uuid.Nil, apperrors.NewConflict("Broadcast template must have a published version")
	}
	return uuid.MustParse(*template.PublishedVersionID), nil
}

func (s *Service) Cancel(ctx context.Context, identifier string) (Broadcast, error) {
	tc, err := requireTenant(ctx, authz.PermissionBroadcastsSend)
	if err != nil {
		return Broadcast{}, err
	}
	id, err := parseID(identifier, "Broadcast id")
	if err != nil {
		return Broadcast{}, err
	}
	value, err := s.repository.Cancel(ctx, tc.Scope.TeamID, id)
	if errors.Is(err, ErrConflict) {
		return Broadcast{}, apperrors.NewConflict("Only scheduled broadcasts can be canceled")
	}
	if err != nil {
		return Broadcast{}, apperrors.NewInternal("Unable to cancel broadcast", err)
	}
	return value, nil
}

func (s *Service) Preview(ctx context.Context, identifier string, req PreviewRequest) (messagetemplate.PreviewResponse, error) {
	value, err := s.Get(ctx, identifier)
	if err != nil {
		return messagetemplate.PreviewResponse{}, err
	}
	return s.templates.Preview(ctx, value.TemplateID, messagetemplate.PreviewRequest{VersionID: pointerValue(value.TemplateVersionID), Variables: req.Variables})
}

func (s *Service) ListRecipients(ctx context.Context, identifier string, req ListRequest) ([]Recipient, error) {
	tc, err := requireTenant(ctx, authz.PermissionBroadcastsRead)
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
	if _, err = s.repository.Get(ctx, tc.Scope.TeamID, id); errors.Is(err, ErrNotFound) {
		return nil, apperrors.NewNotFound("Broadcast not found")
	}
	values, err := s.repository.ListRecipients(ctx, tc.Scope.TeamID, id, req.Limit, req.Offset)
	if err != nil {
		return nil, apperrors.NewInternal("Unable to list broadcast recipients", err)
	}
	return values, nil
}

func (s *Service) GetExclusionSummary(ctx context.Context, identifier string) (ExclusionSummary, error) {
	tc, err := requireTenant(ctx, authz.PermissionBroadcastsRead)
	if err != nil {
		return ExclusionSummary{}, err
	}
	id, err := parseID(identifier, "Broadcast id")
	if err != nil {
		return ExclusionSummary{}, err
	}
	if _, err = s.repository.Get(ctx, tc.Scope.TeamID, id); errors.Is(err, ErrNotFound) {
		return ExclusionSummary{}, apperrors.NewNotFound("Broadcast not found")
	} else if err != nil {
		return ExclusionSummary{}, apperrors.NewInternal("Unable to get broadcast", err)
	}
	summary, err := s.repository.GetExclusionSummary(ctx, tc.Scope.TeamID, id)
	if err != nil {
		return ExclusionSummary{}, apperrors.NewInternal("Unable to summarize excluded recipients", err)
	}
	return summary, nil
}

func (s *Service) GetAnalytics(ctx context.Context, identifier string) (Analytics, error) {
	tc, err := requireTenant(ctx, authz.PermissionBroadcastsRead)
	if err != nil {
		return Analytics{}, err
	}
	id, err := parseID(identifier, "Broadcast id")
	if err != nil {
		return Analytics{}, err
	}
	analytics, err := s.repository.GetAnalytics(ctx, tc.Scope.TeamID, id)
	if errors.Is(err, ErrNotFound) {
		return Analytics{}, apperrors.NewNotFound("Broadcast not found")
	}
	if err != nil {
		return Analytics{}, apperrors.NewInternal("Unable to get broadcast analytics", err)
	}
	return analytics, nil
}

func (s *Service) Duplicate(ctx context.Context, identifier string, req DuplicateRequest) (Broadcast, error) {
	tc, err := requireTenant(ctx, authz.PermissionBroadcastsWrite)
	if err != nil {
		return Broadcast{}, err
	}
	id, err := parseID(identifier, "Broadcast id")
	if err != nil {
		return Broadcast{}, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return Broadcast{}, apperrors.NewBadRequest("Broadcast name is required")
	}
	value, err := s.repository.Duplicate(ctx, tc.Scope.TeamID, id, name)
	if errors.Is(err, ErrNotFound) {
		return Broadcast{}, apperrors.NewNotFound("Broadcast not found")
	}
	if err != nil {
		return Broadcast{}, apperrors.NewInternal("Unable to duplicate broadcast", err)
	}
	return value, nil
}

type broadcastContentTemplateService interface {
	CreateBroadcastContent(context.Context, uuid.UUID, messagetemplate.BroadcastContentRequest) (messagetemplate.Template, error)
	BroadcastPublishedVersion(context.Context, uuid.UUID, string) (string, bool, error)
}

type broadcastContentCleanupService interface {
	DeleteBroadcastContentIfUnreferenced(context.Context, uuid.UUID, string) error
}

func (s *Service) cleanupBroadcastContent(ctx context.Context, teamID uuid.UUID, templateID string) {
	cleanup, ok := s.templates.(broadcastContentCleanupService)
	if !ok {
		return
	}
	_ = cleanup.DeleteBroadcastContentIfUnreferenced(ctx, teamID, templateID)
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
