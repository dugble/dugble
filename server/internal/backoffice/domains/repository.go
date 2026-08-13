package domains

import (
	"context"
	"errors"
	"fmt"
	"time"

	dbsqlc "github.com/coffeyvidzro/dugble/server/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ queries *dbsqlc.Queries }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{queries: dbsqlc.New(db)} }
func (r *Repository) List(ctx context.Context, limit, offset int32) ([]Domain, error) {
	if r == nil || r.queries == nil {
		return nil, errors.New("backoffice domains repository is not configured")
	}
	rows, e := r.queries.BackofficeListDomains(ctx, dbsqlc.BackofficeListDomainsParams{PageLimit: limit, PageOffset: offset})
	if e != nil {
		return nil, fmt.Errorf("list backoffice domains: %w", e)
	}
	out := make([]Domain, 0, len(rows))
	for _, v := range rows {
		out = append(out, domain(v.ID, v.AssetID, v.TeamID, v.TeamName, v.Name, v.OwnerType, v.AssetStatus, v.Provider, v.ProviderAccount, v.Region, v.Status, v.ProviderStatus, v.Verified, v.HealthStatus, v.Attempts, v.ConsecutiveHealthFailures, v.LastError, v.LastCheckedAt.Time, v.LastCheckedAt.Valid, v.NextCheckAt.Time, v.CreatedAt.Time, v.UpdatedAt.Time))
	}
	return out, nil
}
func (r *Repository) Get(ctx context.Context, id uuid.UUID) (Domain, error) {
	v, e := r.queries.BackofficeGetDomain(ctx, dbsqlc.BackofficeGetDomainParams{ID: id})
	if e != nil {
		return Domain{}, fmt.Errorf("get backoffice domain: %w", e)
	}
	return domain(v.ID, v.AssetID, v.TeamID, v.TeamName, v.Name, v.OwnerType, v.AssetStatus, v.Provider, v.ProviderAccount, v.Region, v.Status, v.ProviderStatus, v.Verified, v.HealthStatus, v.Attempts, v.ConsecutiveHealthFailures, v.LastError, v.LastCheckedAt.Time, v.LastCheckedAt.Valid, v.NextCheckAt.Time, v.CreatedAt.Time, v.UpdatedAt.Time), nil
}
func domain(id, assetID, teamID uuid.UUID, teamName, name, ownerType, assetStatus, provider, account, region, status, providerStatus string, verified bool, health string, attempts, failures int32, lastError string, last time.Time, lastValid bool, next, created, updated time.Time) Domain {
	var checked *time.Time
	if lastValid {
		checked = &last
	}
	return Domain{ID: id.String(), AssetID: assetID.String(), TeamID: teamID.String(), TeamName: teamName, Name: name, OwnerType: ownerType, AssetStatus: assetStatus, Provider: provider, ProviderAccount: account, Region: region, Status: status, ProviderStatus: providerStatus, Verified: verified, HealthStatus: health, Attempts: attempts, ConsecutiveHealthFailures: failures, LastError: lastError, LastCheckedAt: checked, NextCheckAt: next, CreatedAt: created, UpdatedAt: updated}
}
