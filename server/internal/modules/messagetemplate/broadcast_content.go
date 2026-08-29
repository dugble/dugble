package messagetemplate

import (
	"context"
	"fmt"
	htmlpkg "html"
	"strings"

	"github.com/google/uuid"

	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

const broadcastTemplateAliasPrefix = "__broadcast_"

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
		return Template{}, apperrors.NewInternal("Unable to publish broadcast content", err)
	}
	return published, nil
}

// BroadcastPublishedVersion returns a pinned version for an internal broadcast
// template without applying public Templates permissions. Broadcast authorization
// is performed by the caller before this method is used.
func (s *Service) BroadcastPublishedVersion(ctx context.Context, teamID uuid.UUID, templateID string) (string, error) {
	if s == nil || s.repository == nil {
		return "", apperrors.NewInternal("Template service is not configured", nil)
	}
	template, err := s.repository.Resolve(ctx, teamID, templateID)
	if err != nil {
		return "", apperrors.NewInternal("Unable to load broadcast content", err)
	}
	if !IsBroadcastTemplate(template) {
		return "", apperrors.NewConflict("Broadcast template is not internal content")
	}
	if template.PublishedVersionID == nil {
		return "", apperrors.NewConflict("Broadcast content has no published version")
	}
	return *template.PublishedVersionID, nil
}

func withPreviewText(body string, previewText *string) string {
	body = strings.TrimSpace(body)
	if previewText == nil || strings.TrimSpace(*previewText) == "" {
		return body
	}
	preview := htmlpkg.EscapeString(strings.TrimSpace(*previewText))
	return fmt.Sprintf(`<div data-dugble-preheader="true" style="display:none;max-height:0;overflow:hidden;opacity:0;color:transparent;">%s</div>%s`, preview, body)
}
