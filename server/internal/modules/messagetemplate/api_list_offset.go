package messagetemplate

import (
	"context"

	"github.com/dugble/dugble/server/internal/authz"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

// ListOffsetAPI returns the public template list using the same limit/offset
// pagination contract as the other list APIs. Broadcast-owned templates are
// implementation details and never consume public pagination positions.
func (s *Service) ListOffsetAPI(ctx context.Context, request ListRequest) (ListResponse, error) {
	access, err := requireAccess(ctx, authz.PermissionTemplatesRead)
	if err != nil {
		return ListResponse{}, err
	}

	normalizeList(&request)
	wanted := int(request.Limit) + 1
	visible := make([]Template, 0, wanted)
	rawOffset := int32(0)
	visibleOffset := int32(0)

	for len(visible) < wanted {
		values, listErr := s.repository.List(ctx, access.Scope.TeamID, 100, rawOffset)
		if listErr != nil {
			return ListResponse{}, apperrors.NewInternal("Unable to list templates", listErr)
		}
		if len(values) == 0 {
			break
		}
		rawOffset += int32(len(values))
		for _, value := range values {
			if IsBroadcastTemplate(value) {
				continue
			}
			if visibleOffset < request.Offset {
				visibleOffset++
				continue
			}
			visible = append(visible, value)
			if len(visible) == wanted {
				break
			}
		}
		if len(values) < 100 {
			break
		}
	}

	hasMore := len(visible) > int(request.Limit)
	if hasMore {
		visible = visible[:request.Limit]
	}

	data := make([]ListItem, 0, len(visible))
	for _, value := range visible {
		data = append(data, ListItem{
			ID:          value.ID,
			Name:        value.Name,
			Category:    value.Category,
			Status:      templateStatus(value),
			PublishedAt: value.PublishedAt,
			CreatedAt:   value.CreatedAt,
			UpdatedAt:   value.UpdatedAt,
			Alias:       value.Alias,
		})
	}

	return ListResponse{Object: ObjectList, Data: data, HasMore: hasMore}, nil
}
