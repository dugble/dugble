package teams

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apperrors "github.com/coffeyvidzro/dugble/server/pkg/errors"
)

type repository interface {
	List(context.Context, Filter) ([]Row, error)
	Detail(context.Context, string) (Detail, error)
	UpdateStatus(context.Context, string, string) error
}

type Service struct{ repository repository }

func NewService(repository repository) *Service { return &Service{repository: repository} }

func (s *Service) List(ctx context.Context, filter Filter) (Page, error) {
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Status = strings.ToLower(strings.TrimSpace(filter.Status))
	if filter.Status != "" && filter.Status != "active" && filter.Status != "disabled" {
		return Page{}, apperrors.NewBadRequest("status must be active or disabled")
	}
	if filter.Offset < 0 {
		return Page{}, apperrors.NewBadRequest("offset must not be negative")
	}
	if filter.Limit == 0 {
		filter.Limit = DefaultPageLimit
	}
	if filter.Limit < 0 || filter.Limit > MaxPageLimit {
		return Page{}, apperrors.NewBadRequest("limit must be between 1 and 100")
	}
	requestedLimit := filter.Limit
	filter.Limit++
	rows, err := s.repository.List(ctx, filter)
	if err != nil {
		return Page{}, err
	}
	hasMore := len(rows) > int(requestedLimit)
	if hasMore {
		rows = rows[:requestedLimit]
	}
	return Page{Data: rows, Limit: requestedLimit, Offset: filter.Offset, HasMore: hasMore}, nil
}

func (s *Service) Detail(ctx context.Context, id string) (Detail, error) {
	id, err := validTeamID(id)
	if err != nil {
		return Detail{}, err
	}
	detail, err := s.repository.Detail(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, apperrors.NewNotFound("Team not found")
	}
	return detail, err
}

func (s *Service) UpdateStatus(ctx context.Context, id string, request StatusRequest) (Detail, error) {
	id, err := validTeamID(id)
	if err != nil {
		return Detail{}, err
	}
	status := strings.ToLower(strings.TrimSpace(request.Status))
	if status != "active" && status != "disabled" {
		return Detail{}, apperrors.NewBadRequest("status must be active or disabled")
	}
	if strings.TrimSpace(request.Reason) == "" {
		return Detail{}, apperrors.NewBadRequest("status change reason is required")
	}
	if err := s.repository.UpdateStatus(ctx, id, status); errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, apperrors.NewNotFound("Team not found")
	} else if err != nil {
		return Detail{}, err
	}
	return s.Detail(ctx, id)
}

func validTeamID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if _, err := uuid.Parse(id); err != nil {
		return "", apperrors.NewBadRequest("team_id must be a valid UUID")
	}
	return id, nil
}
