package senderid

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	dbsqlc "github.com/dugble/dugble/server/internal/database/sqlc"
	"github.com/dugble/dugble/server/pkg/pgconv"
)

var ErrSenderIDAlreadyExists = errors.New("sender id already exists")

type Repository struct {
	queries *dbsqlc.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{queries: dbsqlc.New(db)}
}

func (r *Repository) Create(
	ctx context.Context,
	teamID uuid.UUID,
	name string,
	countryCode string,
	purpose string,
	provider *string,
	createdBy *uuid.UUID,
) (SenderID, error) {
	if r == nil || r.queries == nil {
		return SenderID{}, errors.New("sender id repository is not configured")
	}

	normalizedName := strings.TrimSpace(name)
	countryCode = strings.ToUpper(strings.TrimSpace(countryCode))
	var normalizedProvider *string
	if provider != nil {
		value := strings.ToLower(strings.TrimSpace(*provider))
		if value != "" {
			normalizedProvider = &value
		}
	}

	row, err := r.queries.CreateSenderID(ctx, dbsqlc.CreateSenderIDParams{
		TeamID:      teamID,
		Name:        normalizedName,
		CountryCode: countryCode,
		Purpose:     purpose,
		Provider:    normalizedProvider,
		CreatedBy:   createdBy,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return SenderID{}, ErrSenderIDAlreadyExists
		}
		return SenderID{}, fmt.Errorf("create sender id: %w", err)
	}
	return senderIDFromRow(row), nil
}

func (r *Repository) List(ctx context.Context, teamID uuid.UUID) ([]SenderID, error) {
	if r == nil || r.queries == nil {
		return nil, errors.New("sender id repository is not configured")
	}
	rows, err := r.queries.ListSenderIDs(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("list sender ids: %w", err)
	}
	senders := make([]SenderID, 0, len(rows))
	for _, row := range rows {
		senders = append(senders, senderIDFromRow(row))
	}
	return senders, nil
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID, teamID uuid.UUID) (SenderID, error) {
	if r == nil || r.queries == nil {
		return SenderID{}, errors.New("sender id repository is not configured")
	}
	row, err := r.queries.GetSenderID(ctx, dbsqlc.GetSenderIDParams{ID: id, TeamID: teamID})
	if err != nil {
		return SenderID{}, fmt.Errorf("get sender id: %w", err)
	}
	return senderIDFromRow(row), nil
}

func (r *Repository) Deactivate(ctx context.Context, id uuid.UUID, teamID uuid.UUID) (SenderID, error) {
	if r == nil || r.queries == nil {
		return SenderID{}, errors.New("sender id repository is not configured")
	}
	row, err := r.queries.DeactivateSenderID(ctx, dbsqlc.DeactivateSenderIDParams{ID: id, TeamID: teamID})
	if err != nil {
		return SenderID{}, fmt.Errorf("deactivate sender id: %w", err)
	}
	return senderIDFromRow(row), nil
}

func senderIDFromRow(row dbsqlc.SenderID) SenderID {
	var createdBy *string
	if row.CreatedBy != nil {
		value := row.CreatedBy.String()
		createdBy = &value
	}
	purpose := ""
	if row.Purpose != nil {
		purpose = *row.Purpose
	}
	return SenderID{
		ID:              row.ID.String(),
		TeamID:          row.TeamID.String(),
		Name:            row.Name,
		CountryCode:     row.CountryCode,
		Purpose:         purpose,
		Status:          row.Status,
		Provider:        row.Provider,
		RejectionReason: row.RejectionReason,
		ApprovedAt:      pgconv.TimestamptzToTimePtr(row.ApprovedAt),
		RejectedAt:      pgconv.TimestamptzToTimePtr(row.RejectedAt),
		SuspendedAt:     pgconv.TimestamptzToTimePtr(row.SuspendedAt),
		CreatedBy:       createdBy,
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
