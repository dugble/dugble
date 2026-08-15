package segment

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

func (r *Repository) Create(ctx context.Context, teamID uuid.UUID, name string) (Segment, error) {
	row, err := r.queries.CreateSegment(ctx, dbsqlc.CreateSegmentParams{TeamID: teamID, Name: name})
	if err != nil {
		return Segment{}, fmt.Errorf("create segment: %w", err)
	}
	return segmentFromSQLC(row), nil
}

func (r *Repository) List(ctx context.Context, teamID uuid.UUID, limit, offset int32) ([]Segment, error) {
	rows, err := r.queries.ListSegments(ctx, dbsqlc.ListSegmentsParams{
		TeamID: teamID, PageLimit: limit, PageOffset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list segments: %w", err)
	}
	values := make([]Segment, 0, len(rows))
	for _, row := range rows {
		values = append(values, segmentFromSQLC(row))
	}
	return values, nil
}

func (r *Repository) Get(ctx context.Context, id, teamID uuid.UUID) (Segment, error) {
	row, err := r.queries.GetSegment(ctx, dbsqlc.GetSegmentParams{ID: id, TeamID: teamID})
	if err != nil {
		return Segment{}, err
	}
	return segmentFromSQLC(row), nil
}

func (r *Repository) ListContacts(ctx context.Context, segmentID, teamID uuid.UUID, limit, offset int32) ([]Contact, error) {
	rows, err := r.queries.ListSegmentContacts(ctx, dbsqlc.ListSegmentContactsParams{
		TeamID: teamID, SegmentID: segmentID, PageLimit: limit, PageOffset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list segment contacts: %w", err)
	}
	values := make([]Contact, 0, len(rows))
	for _, row := range rows {
		values = append(values, Contact{
			ID: row.ID.String(), TeamID: row.TeamID.String(), Email: row.Email,
			FirstName: row.FirstName, LastName: row.LastName, Unsubscribed: row.Unsubscribed,
			CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
		})
	}
	return values, nil
}

func (r *Repository) Delete(ctx context.Context, id, teamID uuid.UUID) (Segment, error) {
	row, err := r.queries.DeleteSegment(ctx, dbsqlc.DeleteSegmentParams{ID: id, TeamID: teamID})
	if err != nil {
		return Segment{}, err
	}
	return segmentFromSQLC(row), nil
}

func segmentFromSQLC(row dbsqlc.Segment) Segment {
	return Segment{
		ID:        row.ID.String(),
		TeamID:    row.TeamID.String(),
		Name:      row.Name,
		CreatedAt: row.CreatedAt.Time,
	}
}
