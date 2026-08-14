package renewal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/dugble/dugble/server/internal/billing/subscription/lifecycle"
)

const getDueSQL = `
SELECT id, team_id, status, plan_code, pending_plan_code,
       pending_plan_effective_at, cancel_at_period_end,
       current_period_start, current_period_end
FROM team_subscriptions
WHERE team_id = $1
  AND status IN ('active', 'past_due')
  AND current_period_end <= now()
FOR UPDATE
`

const applyCancellationSQL = `
UPDATE team_subscriptions
SET status = 'canceled', pending_plan_code = NULL,
    pending_plan_effective_at = NULL, cancel_at_period_end = false,
    updated_at = now()
WHERE id = $1
RETURNING current_period_start, current_period_end
`

const applyChargeSQL = `
UPDATE team_subscriptions
SET status = CASE WHEN $2 THEN 'active' ELSE 'past_due' END,
    plan_code = CASE WHEN $2 THEN $3 ELSE plan_code END,
    current_period_start = CASE WHEN $2 THEN $4 ELSE current_period_start END,
    current_period_end = CASE WHEN $2 THEN $5 ELSE current_period_end END,
    pending_plan_code = CASE WHEN $2 THEN NULL ELSE pending_plan_code END,
    pending_plan_effective_at = CASE WHEN $2 THEN NULL ELSE pending_plan_effective_at END,
    updated_at = now()
WHERE id = $1
RETURNING plan_code, current_period_start, current_period_end
`

type Due struct {
	SubscriptionID uuid.UUID
	TeamID         uuid.UUID
	State          lifecycle.State
}

type BillingRecipient struct {
	Name     string
	Email    string
	TeamName string
}

type Repository struct{}

func NewRepository() *Repository { return &Repository{} }

func (*Repository) GetDue(ctx context.Context, tx pgx.Tx, teamID uuid.UUID) (Due, error) {
	if tx == nil {
		return Due{}, errors.New("renew subscription: transaction is required")
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1::text, 0))`, teamID); err != nil {
		return Due{}, fmt.Errorf("lock team billing: %w", err)
	}
	var due Due
	var status string
	var pendingAt pgtype.Timestamptz
	err := tx.QueryRow(ctx, getDueSQL, teamID).Scan(
		&due.SubscriptionID, &due.TeamID, &status, &due.State.PlanCode,
		&due.State.PendingPlanCode, &pendingAt, &due.State.CancelAtPeriodEnd,
		&due.State.CurrentPeriodStart, &due.State.CurrentPeriodEnd,
	)
	if err != nil {
		return Due{}, err
	}
	due.State.Status = lifecycle.Status(status)
	if pendingAt.Valid {
		due.State.PendingPlanEffectiveAt = &pendingAt.Time
	}
	return due, nil
}

func (*Repository) ApplyCancellation(ctx context.Context, tx pgx.Tx, id uuid.UUID) (timeRange, error) {
	var period timeRange
	err := tx.QueryRow(ctx, applyCancellationSQL, id).Scan(&period.Start, &period.End)
	return period, err
}

func (*Repository) ApplyCharge(ctx context.Context, tx pgx.Tx, id uuid.UUID, applied bool, plan string, start, end time.Time) (appliedState, error) {
	var state appliedState
	err := tx.QueryRow(ctx, applyChargeSQL, id, applied, plan, start, end).Scan(&state.Plan, &state.Period.Start, &state.Period.End)
	return state, err
}

func (*Repository) ListBillingRecipients(ctx context.Context, tx pgx.Tx, teamID uuid.UUID) ([]BillingRecipient, error) {
	rows, err := tx.Query(ctx, `
SELECT users.name, users.email, teams.name
FROM team_members
JOIN users ON users.id = team_members.user_id
JOIN teams ON teams.id = team_members.team_id
WHERE team_members.team_id = $1
  AND team_members.role = 'owner'
  AND team_members.status = 'active'
ORDER BY users.id`, teamID)
	if err != nil {
		return nil, fmt.Errorf("list subscription billing recipients: %w", err)
	}
	defer rows.Close()
	recipients := make([]BillingRecipient, 0, 1)
	for rows.Next() {
		var recipient BillingRecipient
		if err := rows.Scan(&recipient.Name, &recipient.Email, &recipient.TeamName); err != nil {
			return nil, fmt.Errorf("scan subscription billing recipient: %w", err)
		}
		recipients = append(recipients, recipient)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list subscription billing recipients: %w", err)
	}
	return recipients, nil
}

type timeRange struct{ Start, End time.Time }
type appliedState struct {
	Plan   string
	Period timeRange
}
