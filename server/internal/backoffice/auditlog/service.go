package auditlog

import (
	"context"
	"errors"
	"strings"

	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

const defaultLimit int32 = 50

type Service struct{ repository *Repository }

func NewService(repository *Repository) *Service { return &Service{repository: repository} }

func (s *Service) List(ctx context.Context, filter Filter) (Page, error) {
	if s == nil || s.repository == nil {
		return Page{}, apperrors.NewInternal("Unable to list audit events", errors.New("audit log service is not configured"))
	}
	filter.Query, filter.Outcome, filter.ActorType = strings.TrimSpace(filter.Query), strings.TrimSpace(filter.Outcome), strings.TrimSpace(filter.ActorType)
	if filter.Limit == 0 {
		filter.Limit = defaultLimit
	}
	if filter.Limit < 1 || filter.Limit > 100 || filter.Offset < 0 {
		return Page{}, apperrors.NewBadRequest("Invalid audit log pagination")
	}
	if filter.Outcome != "" && filter.Outcome != "success" && filter.Outcome != "failure" {
		return Page{}, apperrors.NewBadRequest("Invalid audit outcome")
	}
	if filter.ActorType != "" && filter.ActorType != "user" && filter.ActorType != "team_token" && filter.ActorType != "system" {
		return Page{}, apperrors.NewBadRequest("Invalid actor type")
	}
	events, err := s.repository.List(ctx, filter)
	if err != nil {
		return Page{}, apperrors.NewInternal("Unable to list audit events", err)
	}
	previous := filter.Offset - filter.Limit
	if previous < 0 {
		previous = 0
	}
	return Page{Events: events, Filter: filter, PreviousOffset: previous, NextOffset: filter.Offset + filter.Limit, HasPrevious: filter.Offset > 0, HasNext: len(events) == int(filter.Limit)}, nil
}
