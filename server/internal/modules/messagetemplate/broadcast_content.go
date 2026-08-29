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

func (s *Service) UpdateBroadcastContent(ctx context.Context, teamID uuid.UUID, templateID string, request BroadcastContentRequest) (Template, error) {
	if s == nil || s.repository == nil {
		return Template{}, apperrors.NewInternal("Template service is not configured", nil)
	}
	template, err := s.repository.Resolve(ctx, teamID, templateID)
	if err != nil {
		return Template{}, apperrors.NewInternal("Unable to load broadcast content", err)
	}
	if !IsBroadcastTemplate(template) {
		return Template{}, apperrors.NewConflict("Broadcast does not own its template content")
	}
	if template.CurrentVersionID == nil {
		return Template{}, apperrors.NewConflict("Broadcast content has no current version")
	}
	base, err := s.repository.GetVersion(ctx, teamID, uuid.MustParse(template.ID), uuid.MustParse(*template.CurrentVersionID))
	if err != nil {
		return Template{}, apperrors.NewInternal("Unable to load broadcast content version", err)
	}

	name := strings.TrimSpace(request.Name)
	update := UpdateRequest{BaseVersionID: base.ID}
	if name != "" && name != template.Name {
		update.Name = &name
	}
	if request.Subject != "" {
		update.Subject = &request.Subject
	}
	if request.HTML != "" {
		html := withPreviewText(request.HTML, request.PreviewText)
		update.HTML = &html
	}
	if request.Text != nil {
		update.Text = &request.Text
	}
	if request.FromEmail != nil {
		update.FromEmail = &request.FromEmail
	}
	if request.FromName != nil {
		update.FromName = &request.FromName
	}
	if err := validateUpdate(template, base, &update); err != nil {
		return Template{}, err
	}
	updated, version, err := s.repository.Update(ctx, teamID, template, base, update)
	if err != nil {
		return Template{}, apperrors.NewInternal("Unable to update broadcast content", err)
	}
	published, err := s.repository.Publish(ctx, teamID, uuid.MustParse(updated.ID), uuid.MustParse(version.ID))
	if err != nil {
		return Template{}, apperrors.NewInternal("Unable to publish broadcast content", err)
	}
	return published, nil
}

func withPreviewText(body string, previewText *string) string {
	body = strings.TrimSpace(body)
	if previewText == nil || strings.TrimSpace(*previewText) == "" {
		return body
	}
	preview := htmlpkg.EscapeString(strings.TrimSpace(*previewText))
	return fmt.Sprintf(`<div data-dugble-preheader="true" style="display:none;max-height:0;overflow:hidden;opacity:0;color:transparent;">%s</div>%s`, preview, body)
}
