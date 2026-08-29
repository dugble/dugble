package topic

import (
	"strings"

	"github.com/google/uuid"

	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

const (
	defaultListLimit = int32(50)
	maxListLimit     = int32(100)
	listLookahead    = int32(1)
)

func validateCreate(req CreateRequest) (CreateRequest, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Description = normalizeOptional(req.Description)
	req.DefaultSubscription = strings.ToLower(strings.TrimSpace(req.DefaultSubscription))
	req.Visibility = strings.ToLower(strings.TrimSpace(req.Visibility))
	if req.Visibility == "" {
		req.Visibility = "private"
	}
	if err := validateNameDescription(req.Name, req.Description); err != nil {
		return CreateRequest{}, err
	}
	if req.DefaultSubscription != "opt_in" && req.DefaultSubscription != "opt_out" {
		return CreateRequest{}, apperrors.NewBadRequest("Default subscription must be opt_in or opt_out")
	}
	if req.Visibility != "public" && req.Visibility != "private" {
		return CreateRequest{}, apperrors.NewBadRequest("Visibility must be public or private")
	}
	return req, nil
}

func validateNameDescription(name string, description *string) error {
	if name == "" || len(name) > 50 {
		return apperrors.NewBadRequest("Topic name is required and must be at most 50 characters")
	}
	if description != nil && len(*description) > 200 {
		return apperrors.NewBadRequest("Topic description must be at most 200 characters")
	}
	return nil
}

func normalizeOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func parseID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return uuid.Nil, apperrors.NewBadRequest("Topic id must be a valid UUID")
	}
	return id, nil
}

func normalizeListRequest(req *ListRequest) {
	// The HTTP handler requests one row beyond the public page limit so it can
	// calculate has_more. Preserve that internal lookahead while still bounding
	// all other service callers.
	if req.Limit <= 0 || req.Limit > maxListLimit+listLookahead {
		req.Limit = defaultListLimit
	}
	if req.Offset < 0 {
		req.Offset = 0
	}
}
