package users

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	dbsqlc "github.com/dugble/dugble/server/internal/database/sqlc"
)

type Repository struct {
	queries *dbsqlc.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{queries: dbsqlc.New(db)}
}

func (r *Repository) List(ctx context.Context, filter Filter) ([]Row, error) {
	rows, err := r.queries.BackofficeListUsers(ctx, dbsqlc.BackofficeListUsersParams{
		Search:     filter.Query,
		PageLimit:  filter.Limit,
		PageOffset: filter.Offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	users := make([]Row, 0, len(rows))
	for _, row := range rows {
		users = append(users, userFromListRow(row))
	}
	return users, nil
}

func (r *Repository) Detail(ctx context.Context, id string) (Detail, error) {
	userID, err := uuid.Parse(id)
	if err != nil {
		return Detail{}, fmt.Errorf("parse user id: %w", err)
	}

	user, err := r.queries.BackofficeGetUser(ctx, dbsqlc.BackofficeGetUserParams{ID: userID})
	if err != nil {
		return Detail{}, fmt.Errorf("get user detail: %w", err)
	}

	teamRows, err := r.queries.BackofficeListUserTeams(ctx, dbsqlc.BackofficeListUserTeamsParams{UserID: userID})
	if err != nil {
		return Detail{}, fmt.Errorf("list user teams: %w", err)
	}

	teams := make([]TeamMembershipRow, 0, len(teamRows))
	for _, row := range teamRows {
		teams = append(teams, TeamMembershipRow{
			ID:     row.ID.String(),
			Name:   row.Name,
			Role:   row.Role,
			Status: row.Status,
		})
	}

	return Detail{
		User: Row{
			ID:            user.ID.String(),
			Email:         user.Email,
			Name:          user.Name,
			EmailVerified: user.EmailVerified,
			CreatedAt:     user.CreatedAt.Time,
		},
		Teams: teams,
	}, nil
}

func userFromListRow(row dbsqlc.BackofficeListUsersRow) Row {
	return Row{
		ID:            row.ID.String(),
		Email:         row.Email,
		Name:          row.Name,
		EmailVerified: row.EmailVerified,
		CreatedAt:     row.CreatedAt.Time,
	}
}
