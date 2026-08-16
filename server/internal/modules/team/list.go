package team

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/dugble/dugble/server/internal/authn"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
	"github.com/dugble/dugble/server/pkg/pgconv"
)

const (
	defaultTeamListPage  = 1
	defaultTeamListLimit = 20
	maxTeamListLimit     = 100
)

type ListOptions struct {
	Page   int
	Limit  int
	Search string
	Status string
}

type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type ListMeta struct {
	Pagination PaginationMeta `json:"pagination"`
}

func parseListOptions(pageValue, limitValue, searchValue, statusValue string) (ListOptions, error) {
	options := ListOptions{
		Page:   defaultTeamListPage,
		Limit:  defaultTeamListLimit,
		Search: strings.TrimSpace(searchValue),
		Status: TeamStatusActive,
	}

	if value := strings.TrimSpace(pageValue); value != "" {
		page, err := strconv.Atoi(value)
		if err != nil || page < 1 {
			return ListOptions{}, apperrors.NewBadRequest("Page must be a positive integer")
		}
		options.Page = page
	}

	if value := strings.TrimSpace(limitValue); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil || limit < 1 || limit > maxTeamListLimit {
			return ListOptions{}, apperrors.NewBadRequest("Limit must be between 1 and 100")
		}
		options.Limit = limit
	}

	if value := strings.ToLower(strings.TrimSpace(statusValue)); value != "" {
		if value != TeamStatusActive && value != TeamStatusDisabled {
			return ListOptions{}, apperrors.NewBadRequest("Status must be active or disabled")
		}
		options.Status = value
	}

	return options, nil
}

func (s *Service) ListPaginated(ctx context.Context, options ListOptions) ([]Team, ListMeta, error) {
	principal, ok := authn.PrincipalFromContext(ctx)
	if !ok {
		return nil, ListMeta{}, apperrors.NewUnauthorized("Authentication is required")
	}
	teams, total, err := s.repository.ListForUserPaginated(ctx, principal.UserID, options)
	if err != nil {
		return nil, ListMeta{}, apperrors.NewInternal("Unable to list teams", err)
	}
	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(options.Limit) - 1) / int64(options.Limit))
	}
	return teams, ListMeta{Pagination: PaginationMeta{
		Page:       options.Page,
		Limit:      options.Limit,
		Total:      total,
		TotalPages: totalPages,
	}}, nil
}

func (r *Repository) ListForUserPaginated(ctx context.Context, userID uuid.UUID, options ListOptions) ([]Team, int64, error) {
	searchPattern := "%" + options.Search + "%"
	var total int64
	if err := r.db.QueryRow(ctx, `
		SELECT count(*)
		FROM teams AS t
		JOIN team_members AS tm ON tm.team_id = t.id
		WHERE tm.user_id = $1
		  AND tm.status = 'active'
		  AND t.status = $2
		  AND ($3 = '' OR t.name ILIKE $4)`, userID, options.Status, options.Search, searchPattern).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count teams for user: %w", err)
	}

	offset := (options.Page - 1) * options.Limit
	rows, err := r.db.Query(ctx, `
		SELECT t.id, t.name, t.market_code, t.phone, t.address, t.website, t.status, t.created_by, t.created_at, t.updated_at, tm.role
		FROM teams AS t
		JOIN team_members AS tm ON tm.team_id = t.id
		WHERE tm.user_id = $1
		  AND tm.status = 'active'
		  AND t.status = $2
		  AND ($3 = '' OR t.name ILIKE $4)
		ORDER BY t.created_at DESC, t.id DESC
		LIMIT $5 OFFSET $6`, userID, options.Status, options.Search, searchPattern, options.Limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list teams for user: %w", err)
	}
	defer rows.Close()

	teams := make([]Team, 0, options.Limit)
	for rows.Next() {
		var (
			team      Team
			teamID    uuid.UUID
			createdBy *uuid.UUID
			createdAt pgtype.Timestamptz
			updatedAt pgtype.Timestamptz
		)
		if err := rows.Scan(
			&teamID, &team.Name, &team.MarketCode, &team.Phone, &team.Address, &team.Website,
			&team.Status, &createdBy, &createdAt, &updatedAt, &team.UserRole,
		); err != nil {
			return nil, 0, fmt.Errorf("scan team: %w", err)
		}
		team.ID = teamID.String()
		team.CreatedBy = stringPointer(createdBy)
		team.CreatedAt = pgconv.TimestamptzToTime(createdAt)
		team.UpdatedAt = pgconv.TimestamptzToTime(updatedAt)
		teams = append(teams, team)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate teams: %w", err)
	}
	return teams, total, nil
}
