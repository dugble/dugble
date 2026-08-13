package subscription

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	dbsqlc "github.com/coffeyvidzro/dugble/server/internal/database/sqlc"
)

const listChargesWithCreditSQL = `
SELECT
    charge.id::text,
    charge.subscription_id::text,
    charge.plan_price_id::text,
    charge.plan_code,
    charge.billing_market::text,
    charge.currency::text,
    charge.period_start,
    charge.period_end,
    charge.amount_units,
    charge.status,
    charge.failure_code,
    charge.attempt_count,
    charge.last_attempted_at,
    charge.applied_at,
    charge.reference_id,
    credit.id::text,
    credit.granted_units,
    credit.consumed_units,
    CASE
        WHEN credit.id IS NULL THEN NULL
        ELSE GREATEST(credit.granted_units - credit.consumed_units, 0)
    END,
    charge.created_at
FROM subscription_charges AS charge
LEFT JOIN subscription_credits AS credit
  ON credit.subscription_charge_id = charge.id
WHERE charge.team_id = $1
ORDER BY charge.created_at DESC, charge.id DESC
LIMIT $2
OFFSET $3
`

type Repository struct {
	db      *pgxpool.Pool
	queries *dbsqlc.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db, queries: dbsqlc.New(db)}
}

func (r *Repository) GetSubscription(ctx context.Context, teamID uuid.UUID) (Subscription, error) {
	row, err := r.queries.GetTeamSubscription(ctx, dbsqlc.GetTeamSubscriptionParams{TeamID: teamID})
	if err != nil {
		return Subscription{}, err
	}
	return subscriptionFromSQLC(row), nil
}

func (r *Repository) ListCharges(ctx context.Context, teamID uuid.UUID, limit, offset int32) ([]Charge, error) {
	rows, err := r.db.Query(ctx, listChargesWithCreditSQL, teamID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	charges := make([]Charge, 0)
	for rows.Next() {
		var charge Charge
		var failureCode, creditID pgtype.Text
		var appliedAt pgtype.Timestamptz
		var grantedUnits, consumedUnits, remainingUnits pgtype.Int8
		if err := rows.Scan(
			&charge.ID,
			&charge.SubscriptionID,
			&charge.PlanPriceID,
			&charge.PlanCode,
			&charge.BillingMarket,
			&charge.Currency,
			&charge.PeriodStart,
			&charge.PeriodEnd,
			&charge.AmountUnits,
			&charge.Status,
			&failureCode,
			&charge.AttemptCount,
			&charge.LastAttemptedAt,
			&appliedAt,
			&charge.ReferenceID,
			&creditID,
			&grantedUnits,
			&consumedUnits,
			&remainingUnits,
			&charge.CreatedAt,
		); err != nil {
			return nil, err
		}
		if failureCode.Valid {
			charge.FailureCode = &failureCode.String
		}
		if appliedAt.Valid {
			value := appliedAt.Time
			charge.AppliedAt = &value
		}
		if creditID.Valid && grantedUnits.Valid && consumedUnits.Valid && remainingUnits.Valid {
			charge.CommunicationCredit = &CommunicationCredit{
				ID:             creditID.String,
				GrantedUnits:   grantedUnits.Int64,
				ConsumedUnits:  consumedUnits.Int64,
				RemainingUnits: remainingUnits.Int64,
			}
		}
		charges = append(charges, charge)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return charges, nil
}

func (r *Repository) SchedulePlanChange(ctx context.Context, teamID uuid.UUID, plan string) (Subscription, error) {
	row, err := r.queries.ScheduleTeamPlanChange(ctx, dbsqlc.ScheduleTeamPlanChangeParams{
		TeamID: teamID, PlanCode: plan,
	})
	if err != nil {
		return Subscription{}, err
	}
	return subscriptionFromScheduleRow(row), nil
}

func (r *Repository) CancelPlanChange(ctx context.Context, teamID uuid.UUID) (Subscription, error) {
	row, err := r.queries.CancelTeamPlanChange(ctx, dbsqlc.CancelTeamPlanChangeParams{TeamID: teamID})
	if err != nil {
		return Subscription{}, err
	}
	return subscriptionFromCancelRow(row), nil
}

func (r *Repository) Cancel(ctx context.Context, teamID uuid.UUID) (Subscription, error) {
	row, err := r.queries.CancelTeamSubscription(ctx, dbsqlc.CancelTeamSubscriptionParams{TeamID: teamID})
	if err != nil {
		return Subscription{}, err
	}
	return subscriptionFromCancelSubscriptionRow(row), nil
}

func (r *Repository) Reactivate(ctx context.Context, teamID uuid.UUID) (Subscription, error) {
	row, err := r.queries.ReactivateTeamSubscription(ctx, dbsqlc.ReactivateTeamSubscriptionParams{TeamID: teamID})
	if err != nil {
		return Subscription{}, err
	}
	return subscriptionFromSQLC(row), nil
}

func subscriptionFromSQLC(row dbsqlc.TeamSubscription) Subscription {
	return subscriptionFromFields(
		row.ID,
		row.TeamID,
		row.PlanCode,
		row.Status,
		row.CurrentPeriodStart,
		row.CurrentPeriodEnd,
		row.PendingPlanCode,
		row.PendingPlanEffectiveAt,
		row.CancelAtPeriodEnd,
		row.CreatedAt,
		row.UpdatedAt,
	)
}

func subscriptionFromCancelSubscriptionRow(row dbsqlc.CancelTeamSubscriptionRow) Subscription {
	return subscriptionFromFields(
		row.ID, row.TeamID, row.PlanCode, row.Status,
		row.CurrentPeriodStart, row.CurrentPeriodEnd,
		row.PendingPlanCode, row.PendingPlanEffectiveAt,
		row.CancelAtPeriodEnd, row.CreatedAt, row.UpdatedAt,
	)
}

func subscriptionFromScheduleRow(row dbsqlc.ScheduleTeamPlanChangeRow) Subscription {
	return subscriptionFromFields(
		row.ID,
		row.TeamID,
		row.PlanCode,
		row.Status,
		row.CurrentPeriodStart,
		row.CurrentPeriodEnd,
		row.PendingPlanCode,
		row.PendingPlanEffectiveAt,
		row.CancelAtPeriodEnd,
		row.CreatedAt,
		row.UpdatedAt,
	)
}

func subscriptionFromCancelRow(row dbsqlc.CancelTeamPlanChangeRow) Subscription {
	return subscriptionFromFields(
		row.ID,
		row.TeamID,
		row.PlanCode,
		row.Status,
		row.CurrentPeriodStart,
		row.CurrentPeriodEnd,
		row.PendingPlanCode,
		row.PendingPlanEffectiveAt,
		row.CancelAtPeriodEnd,
		row.CreatedAt,
		row.UpdatedAt,
	)
}

func subscriptionFromFields(
	id uuid.UUID,
	teamID uuid.UUID,
	planCode string,
	status string,
	periodStart pgtype.Timestamptz,
	periodEnd pgtype.Timestamptz,
	pendingPlanCode *string,
	pendingPlanEffectiveAt pgtype.Timestamptz,
	cancelAtPeriodEnd bool,
	createdAt pgtype.Timestamptz,
	updatedAt pgtype.Timestamptz,
) Subscription {
	var pendingEffectiveAt *time.Time
	if pendingPlanEffectiveAt.Valid {
		value := pendingPlanEffectiveAt.Time
		pendingEffectiveAt = &value
	}
	return Subscription{
		ID:                     id.String(),
		TeamID:                 teamID.String(),
		PlanCode:               planCode,
		Status:                 status,
		CurrentPeriodStart:     periodStart.Time,
		CurrentPeriodEnd:       periodEnd.Time,
		PendingPlanCode:        pendingPlanCode,
		PendingPlanEffectiveAt: pendingEffectiveAt,
		CancelAtPeriodEnd:      cancelAtPeriodEnd,
		CreatedAt:              createdAt.Time,
		UpdatedAt:              updatedAt.Time,
	}
}
