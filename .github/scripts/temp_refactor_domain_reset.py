from pathlib import Path
import re

path = Path('server/internal/modules/domain/repository.go')
s = path.read_text()
pattern = r'''func \(r \*Repository\) ResetReconciliationAttempts\(ctx context\.Context, id uuid\.UUID\) error \{.*?\n\}'''
replacement = '''func (r *Repository) ResetReconciliationAttempts(ctx context.Context, id uuid.UUID) error {
	if err := r.requireConfigured(); err != nil {
		return err
	}
	rows, err := r.queries.ResetDomainReconciliationAttempts(ctx, dbsqlc.ResetDomainReconciliationAttemptsParams{ID: id})
	if err != nil {
		return fmt.Errorf("reset sender domain reconciliation attempts: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("reset sender domain reconciliation attempts: sender domain not found")
	}
	return nil
}'''
s, count = re.subn(pattern, replacement, s, count=1, flags=re.S)
if count != 1:
    raise SystemExit('ResetReconciliationAttempts method not found')
path.write_text(s)
