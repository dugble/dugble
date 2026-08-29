package messagetemplate

import (
	"context"

	"github.com/dugble/dugble/server/internal/authz"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

// ListOffsetAPI returns the public template list using the same limit/offset
// pagination contract as the other list APIs.
func (s *Service) ListOffsetAPI(ctx context.Context, request ListRequest) (ListResponse, error) {
	access, err := requireAccess(ctx, authz.PermissionTemplatesRead)
	if err != nil {
		return ListResponse{}, err
	}

	normalizeList(&request)
	values, err := s.repository.List(ctx, access.Scope.TeamID, request.Limit+1, request.Offset)
	if err != nil {
		return ListResponse{}, apperrors.NewInternal("Unable to list templates", err)
	}

	hasMore := len(values) > int(request.Limit)
	if hasMore {
		values = values[:request.Limit]
	}

	data := make([]ListItem, 0, len(values))
	for _, value := range values {
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
