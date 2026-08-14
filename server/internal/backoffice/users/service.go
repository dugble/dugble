package users

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

type repository interface {
	List(context.Context, Filter) ([]Row, error)
	Detail(context.Context, string) (Detail, error)
}

type Service struct{ repository repository }

func NewService(repository repository) *Service { return &Service{repository: repository} }

func (s *Service) List(ctx context.Context, filter Filter) (Page, error) {
	filter.Query = strings.TrimSpace(filter.Query)
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
	id = strings.TrimSpace(id)
	if _, err := uuid.Parse(id); err != nil {
		return Detail{}, apperrors.NewBadRequest("user_id must be a valid UUID")
	}
	detail, err := s.repository.Detail(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, apperrors.NewNotFound("User not found")
	}
	return detail, err
}
