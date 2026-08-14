package dashboard

import (
	"context"
	"fmt"

	dbsqlc "github.com/dugble/dugble/server/internal/database/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ queries *dbsqlc.Queries }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{queries: dbsqlc.New(db)} }
func (r *Repository) Stats(ctx context.Context) (Stats, error) {
	row, err := r.queries.BackofficeDashboardStats(ctx)
	if err != nil {
		return Stats{}, fmt.Errorf("load dashboard stats: %w", err)
	}
	return Stats{Users: row.Users, Teams: row.Teams, SMSToday: row.SmsToday, FailedSMS24Hours: row.FailedSms24Hours, PendingSenderIDs: row.PendingSenderIds, PendingDomains: row.PendingDomains}, nil
}

func (r *Repository) Operations(ctx context.Context) (Operations, error) {
	stats, err := r.Stats(ctx)
	if err != nil {
		return Operations{}, err
	}
	failedRows, err := r.queries.BackofficeDashboardFailedSMS(ctx)
	if err != nil {
		return Operations{}, fmt.Errorf("load failed SMS queue: %w", err)
	}
	senderRows, err := r.queries.BackofficeDashboardPendingSenderIDs(ctx)
	if err != nil {
		return Operations{}, fmt.Errorf("load pending sender ID queue: %w", err)
	}
	domainRows, err := r.queries.BackofficeDashboardPendingDomains(ctx)
	if err != nil {
		return Operations{}, fmt.Errorf("load pending domain queue: %w", err)
	}
	activityRows, err := r.queries.BackofficeDashboardRecentActivity(ctx)
	if err != nil {
		return Operations{}, fmt.Errorf("load recent activity: %w", err)
	}

	result := Operations{Stats: stats}
	for _, row := range failedRows {
		result.FailedSMS = append(result.FailedSMS, FailedSMS{ID: row.ID.String(), TeamName: row.TeamName, ToNumber: row.ToNumber, Status: row.Status, ErrorMessage: row.ErrorMessage, CreatedAt: row.CreatedAt.Time})
	}
	for _, row := range senderRows {
		result.PendingSenderIDs = append(result.PendingSenderIDs, PendingSenderID{ID: row.ID.String(), TeamName: row.TeamName, Name: row.Name, CountryCode: row.CountryCode, CreatedAt: row.CreatedAt.Time})
	}
	for _, row := range domainRows {
		result.PendingDomains = append(result.PendingDomains, PendingDomain{ID: row.ID.String(), TeamName: row.TeamName, Name: row.Name, CreatedAt: row.CreatedAt.Time})
	}
	for _, row := range activityRows {
		result.RecentActivity = append(result.RecentActivity, Activity{Action: row.Action, ResourceType: row.ResourceType, ResourceID: row.ResourceID, Outcome: row.Outcome, ActorType: row.ActorType, CreatedAt: row.CreatedAt.Time})
	}
	return result, nil
}
