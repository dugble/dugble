package messagetemplate

import (
	"context"
	"errors"
	"fmt"
	htmlpkg "html"
	"strings"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

const (
	broadcastTemplateAliasPrefix = "__broadcast_"
	broadcastCleanupTimeout      = 5 * time.Second
)

// BroadcastContentRequest is the internal bridge between the broadcast composer
// and the existing versioned template renderer. Templates created through this
// path are implementation details and are not intended to be reusable resources.
type BroadcastContentRequest struct {
	Name        string
	Subject     string
	HTML        string
	Text        *string
	FromEmail   *string
	FromName    *string
	PreviewText *string
}

func IsBroadcastTemplate(value Template) bool {
	return value.Alias != nil && strings.HasPrefix(*value.Alias, broadcastTemplateAliasPrefix)
}

func (s *Service) CreateBroadcastContent(ctx context.Context, teamID uuid.UUID, request BroadcastContentRequest) (Template, error) {
	if s == nil || s.repository == nil {
		return Template{}, apperrors.NewInternal("Template service is not configured", nil)
	}
	alias := broadcastTemplateAliasPrefix + strings.ReplaceAll(uuid.NewString(), "-", "_")
	name := strings.TrimSpace(request.Name)
	if name == "" {
		name = "Broadcast content"
	}
	create, err := validateCreate(CreateRequest{
		Name: name, Alias: &alias, Category: CategoryCustom,
		FromEmail: request.FromEmail, FromName: request.FromName,
		Subject: request.Subject, HTML: withPreviewText(request.HTML, request.PreviewText), Text: request.Text,
	})
	if err != nil {
		return Template{}, err
	}
	template, version, err := s.repository.Create(ctx, teamID, create)
	if err != nil {
		return Template{}, apperrors.NewInternal("Unable to create broadcast content", err)
	}
	published, err := s.repository.Publish(ctx, teamID, uuid.MustParse(template.ID), uuid.MustParse(version.ID))
	if err != nil {
		if cleanupErr := s.DeleteBroadcastContentIfUnreferenced(ctx, teamID, template.ID); cleanupErr != nil {
			err = errors.Join(err, cleanupErr)
		}
		return Template{}, apperrors.NewInternal("Unable to publish broadcast content", err)
	}
	return published, nil
}

// DeleteBroadcastContentIfUnreferenced hard-deletes an internal broadcast
// template only when no broadcast still references it. Cleanup deliberately
// ignores request cancellation for a short bounded period so a canceled or
// conflicted request does not leave an internal template behind.
func (s *Service) DeleteBroadcastContentIfUnreferenced(ctx context.Context, teamID uuid.UUID, templateID string) error {
	if s == nil || s.repository == nil {
		return errors.New("template service is not configured")
	}
	id, err := uuid.Parse(templateID)
	if err != nil {
		return fmt.Errorf("parse broadcast content template id: %w", err)
	}

	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), broadcastCleanupTimeout)
	defer cancel()
	_, err = s.repository.db.Exec(cleanupCtx, `
DELETE FROM message_templates AS mt
WHERE mt.id = $1
  AND mt.team_id = $2
  AND mt.alias LIKE $3
  AND NOT EXISTS (
      SELECT 1
      FROM broadcasts AS b
      WHERE b.template_id = mt.id
  )`, id, teamID, broadcastTemplateAliasPrefix+"%")
	if err != nil {
		return fmt.Errorf("delete unreferenced broadcast content: %w", err)
	}
	return nil
}

// BroadcastPublishedVersion identifies internal broadcast content and returns
// its published version without applying public Templates permissions. The
// caller is responsible for Broadcast authorization.
func (s *Service) BroadcastPublishedVersion(ctx context.Context, teamID uuid.UUID, templateID string) (string, bool, error) {
	if s == nil || s.repository == nil {
		return "", false, apperrors.NewInternal("Template service is not configured", nil)
	}
	template, err := s.repository.Resolve(ctx, teamID, templateID)
	if err != nil {
		return "", false, apperrors.NewInternal("Unable to load broadcast content", err)
	}
	if !IsBroadcastTemplate(template) {
		return "", false, nil
	}
	if template.PublishedVersionID == nil {
		return "", true, apperrors.NewConflict("Broadcast content has no published version")
	}
	return *template.PublishedVersionID, true, nil
}

func withPreviewText(body string, previewText *string) string {
	body = strings.TrimSpace(body)
	if previewText == nil || strings.TrimSpace(*previewText) == "" {
		return body
	}
	preview := htmlpkg.EscapeString(strings.TrimSpace(*previewText))
	return fmt.Sprintf(`<div data-dugble-preheader='true' style='display:none;max-height:0;overflow:hidden;opacity:0;color:transparent;'>%s</div>%s`, preview, body)
}
