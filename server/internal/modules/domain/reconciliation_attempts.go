package domain

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// ResetReconciliationAttempts clears accumulated reconciliation attempts after a
// successful provider/DNS check. Pending verification is a healthy state and
// must not make a later transient error look like a high-order retry.
func (r *Repository) ResetReconciliationAttempts(ctx context.Context, id uuid.UUID) error {
	if err := r.requireConfigured(); err != nil {
		return err
	}
	result, err := r.db.Exec(ctx, `
UPDATE domains
SET reconciliation_attempts = 0,
    updated_at = now()
WHERE id = $1
`, id)
	if err != nil {
		return fmt.Errorf("reset sender domain reconciliation attempts: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("reset sender domain reconciliation attempts: sender domain not found")
	}
	return nil
}
