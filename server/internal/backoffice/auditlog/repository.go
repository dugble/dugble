package auditlog

import (
	"context"
	"fmt"

	dbsqlc "github.com/dugble/dugble/server/internal/database/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ queries *dbsqlc.Queries }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{queries: dbsqlc.New(db)} }

func (r *Repository) List(ctx context.Context, filter Filter) ([]Event, error) {
	rows, err := r.queries.BackofficeListAuditEvents(ctx, dbsqlc.BackofficeListAuditEventsParams{Search: filter.Query, Outcome: filter.Outcome, ActorType: filter.ActorType, PageLimit: filter.Limit, PageOffset: filter.Offset})
	if err != nil {
		return nil, fmt.Errorf("list backoffice audit events: %w", err)
	}
	events := make([]Event, 0, len(rows))
	for _, row := range rows {
		requestID, ip := "", ""
		if row.RequestID != nil {
			requestID = *row.RequestID
		}
		if row.IpAddress != nil {
			ip = *row.IpAddress
		}
		events = append(events, Event{ID: row.ID.String(), Action: row.Action, ResourceType: row.ResourceType, ResourceID: row.ResourceID, Outcome: row.Outcome, ActorType: row.ActorType, ActorEmail: row.ActorEmail, TeamName: row.TeamName, RequestID: requestID, IPAddress: ip, Metadata: string(row.Metadata), CreatedAt: row.CreatedAt.Time})
	}
	return events, nil
}
