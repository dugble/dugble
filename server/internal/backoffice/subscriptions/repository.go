package subscriptions

import (
	"context"
	"fmt"
	"time"

	dbsqlc "github.com/dugble/dugble/server/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db      *pgxpool.Pool
	queries *dbsqlc.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db, queries: dbsqlc.New(db)} }
func (r *Repository) List(ctx context.Context, f Filter, teamID *uuid.UUID) ([]Subscription, error) {
	rows, e := r.queries.BackofficeListSubscriptions(ctx, dbsqlc.BackofficeListSubscriptionsParams{TeamID: teamID, Status: f.Status, PageLimit: f.Limit, PageOffset: f.Offset})
	if e != nil {
		return nil, fmt.Errorf("list subscriptions: %w", e)
	}
	out := make([]Subscription, 0, len(rows))
	for _, v := range rows {
		var pendingAt *time.Time
		if v.PendingPlanEffectiveAt.Valid {
			x := v.PendingPlanEffectiveAt.Time
			pendingAt = &x
		}
		out = append(out, Subscription{v.ID.String(), v.TeamID.String(), v.TeamName, v.PlanCode, v.Status, v.BillingMarket, v.Currency, v.CurrentPeriodStart.Time, v.CurrentPeriodEnd.Time, v.PendingPlanCode, pendingAt, v.CancelAtPeriodEnd, v.CreatedAt.Time, v.UpdatedAt.Time})
	}
	return out, nil
}
func (r *Repository) Get(ctx context.Context, id uuid.UUID) (Subscription, error) {
	v, e := r.queries.BackofficeGetSubscription(ctx, dbsqlc.BackofficeGetSubscriptionParams{ID: id})
	if e != nil {
		return Subscription{}, e
	}
	var pendingAt *time.Time
	if v.PendingPlanEffectiveAt.Valid {
		x := v.PendingPlanEffectiveAt.Time
		pendingAt = &x
	}
	return Subscription{v.ID.String(), v.TeamID.String(), v.TeamName, v.PlanCode, v.Status, v.BillingMarket, v.Currency, v.CurrentPeriodStart.Time, v.CurrentPeriodEnd.Time, v.PendingPlanCode, pendingAt, v.CancelAtPeriodEnd, v.CreatedAt.Time, v.UpdatedAt.Time}, nil
}
func (r *Repository) Charges(ctx context.Context, id uuid.UUID, limit, offset int32) ([]Charge, error) {
	rows, e := r.queries.BackofficeListSubscriptionCharges(ctx, dbsqlc.BackofficeListSubscriptionChargesParams{SubscriptionID: id, PageLimit: limit, PageOffset: offset})
	if e != nil {
		return nil, fmt.Errorf("list subscription charges: %w", e)
	}
	out := make([]Charge, 0, len(rows))
	for _, v := range rows {
		out = append(out, Charge{v.ID.String(), v.SubscriptionID.String(), v.TeamID.String(), v.PlanCode, v.BillingMarket, v.Currency, v.PeriodStart.Time, v.PeriodEnd.Time, v.AmountUnits, v.Status, v.FailureCode, v.AttemptCount, v.ReferenceID, v.CreatedAt.Time})
	}
	return out, nil
}

func (r *Repository) ChangePlan(ctx context.Context, subscriptionID, teamID uuid.UUID, plan, reason string, actorID uuid.UUID, sessionID string) error {
	return r.lifecycle(ctx, subscriptionID, teamID, "subscription.change_plan", reason, actorID, sessionID, map[string]any{"plan_code": plan}, func(db dbsqlc.DBTX) error {
		q := dbsqlc.New(db)
		_, err := q.ScheduleTeamPlanChange(ctx, dbsqlc.ScheduleTeamPlanChangeParams{TeamID: teamID, PlanCode: plan})
		return err
	})
}
func (r *Repository) CancelPlanChange(ctx context.Context, subscriptionID, teamID uuid.UUID, reason string, actorID uuid.UUID, sessionID string) error {
	return r.lifecycle(ctx, subscriptionID, teamID, "subscription.cancel_plan_change", reason, actorID, sessionID, nil, func(db dbsqlc.DBTX) error {
		result, err := db.Exec(ctx, `UPDATE team_subscriptions SET pending_plan_code=NULL,pending_plan_effective_at=NULL,updated_at=now() WHERE team_id=$1 AND status IN ('active','past_due') AND pending_plan_code IS NOT NULL`, teamID)
		if err == nil && result.RowsAffected() != 1 {
			return pgx.ErrNoRows
		}
		return err
	})
}
func (r *Repository) Cancel(ctx context.Context, subscriptionID, teamID uuid.UUID, reason string, actorID uuid.UUID, sessionID string) error {
	return r.lifecycle(ctx, subscriptionID, teamID, "subscription.cancel", reason, actorID, sessionID, nil, func(db dbsqlc.DBTX) error {
		q := dbsqlc.New(db)
		_, err := q.CancelTeamSubscription(ctx, dbsqlc.CancelTeamSubscriptionParams{TeamID: teamID})
		return err
	})
}
func (r *Repository) Reactivate(ctx context.Context, subscriptionID, teamID uuid.UUID, reason string, actorID uuid.UUID, sessionID string) error {
	return r.lifecycle(ctx, subscriptionID, teamID, "subscription.reactivate", reason, actorID, sessionID, nil, func(db dbsqlc.DBTX) error {
		q := dbsqlc.New(db)
		_, err := q.ReactivateTeamSubscription(ctx, dbsqlc.ReactivateTeamSubscriptionParams{TeamID: teamID})
		return err
	})
}
func (r *Repository) lifecycle(ctx context.Context, subscriptionID, teamID uuid.UUID, action, reason string, actorID uuid.UUID, sessionID string, extra map[string]any, update func(dbsqlc.DBTX) error) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin subscription operation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := update(tx); err != nil {
		return err
	}
	metadata := map[string]any{"reason": reason}
	for key, value := range extra {
		metadata[key] = value
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_events(team_id,actor_type,actor_user_id,actor_session_id,action,resource_type,resource_id,metadata) VALUES($1,'user',$2,NULLIF($3,''),$4,'subscription',$5,$6)`, teamID, actorID, sessionID, action, subscriptionID.String(), metadata)
	if err != nil {
		return fmt.Errorf("audit subscription operation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit subscription operation: %w", err)
	}
	return nil
}
