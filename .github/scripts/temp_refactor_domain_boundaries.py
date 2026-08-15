from pathlib import Path

repo_path = Path('server/internal/modules/domain/repository.go')
s = repo_path.read_text()
s = s.replace('type Repository struct {\n\tdb      *pgxpool.Pool\n\tqueries *dbsqlc.Queries\n}\n\nfunc NewRepository(db *pgxpool.Pool) *Repository {\n\treturn &Repository{db: db, queries: dbsqlc.New(db)}\n}\n', 'type Repository struct {\n\tqueries *dbsqlc.Queries\n}\n\nfunc NewRepository(db *pgxpool.Pool) *Repository {\n\treturn &Repository{queries: dbsqlc.New(db)}\n}\n\nfunc (r *Repository) WithTx(tx pgx.Tx) *Repository {\n\treturn &Repository{queries: r.queries.WithTx(tx)}\n}\n')
for message in [
    'begin sender domain creation',
    'begin sender domain verification update',
    'begin sender domain reconciliation completion',
    'begin sender domain manual health update',
]:
    block = f'''\ttx, err := r.db.BeginTx(ctx, pgx.TxOptions{{}})\n\tif err != nil {{\n\t\treturn SenderDomain{{}}, fmt.Errorf("{message}: %w", err)\n\t}}\n\tdefer func() {{ _ = tx.Rollback(ctx) }}()\n\n\tqueries := r.queries.WithTx(tx)\n'''
    if block not in s:
        raise SystemExit(f'missing repository begin block: {message}')
    s = s.replace(block, '\tqueries := r.queries\n', 1)
for message in [
    'commit sender domain creation',
    'commit sender domain verification',
    'commit sender domain reconciliation completion',
    'commit sender domain manual health update',
]:
    block = f'''\tif err := tx.Commit(ctx); err != nil {{\n\t\treturn SenderDomain{{}}, fmt.Errorf("{message}: %w", err)\n\t}}\n'''
    if block not in s:
        raise SystemExit(f'missing repository commit block: {message}')
    s = s.replace(block, '', 1)
s = s.replace('if r == nil || r.db == nil || r.queries == nil {', 'if r == nil || r.queries == nil {')
repo_path.write_text(s)

service_path = Path('server/internal/modules/domain/service.go')
s = service_path.read_text()
s = s.replace('"github.com/google/uuid"\n', '"github.com/google/uuid"\n\t"github.com/jackc/pgx/v5"\n\t"github.com/jackc/pgx/v5/pgxpool"\n')
s = s.replace('type Service struct {\n\trepository      *Repository', 'type Service struct {\n\tdb              *pgxpool.Pool\n\trepository      *Repository')
marker = 'func (s *Service) WithNotifier(notifier statusNotifier) *Service {\n\ts.notifier = notifier\n\treturn s\n}\n'
replacement = marker + '\nfunc (s *Service) WithDatabase(db *pgxpool.Pool) *Service {\n\tif s != nil {\n\t\ts.db = db\n\t}\n\treturn s\n}\n'
if marker not in s:
    raise SystemExit('WithNotifier marker missing')
s = s.replace(marker, replacement, 1)
s = s.replace('s.repository.Create(ctx, CreateDomainInput{', 's.createDomain(ctx, CreateDomainInput{')
s = s.replace('s.repository.UpdateVerification(', 's.updateVerification(')
s = s.replace('s.repository.UpdateManualHealthCheck(', 's.updateManualHealthCheck(')
helper_marker = '\nfunc requireTenantPermission(ctx context.Context, permission authz.Permission) (authz.Access, error) {'
helpers = '''
func (s *Service) withRepositoryTx(
\tctx context.Context,
\toperation string,
\tfn func(*Repository) (SenderDomain, error),
) (SenderDomain, error) {
\tif s == nil || s.db == nil || s.repository == nil {
\t\treturn SenderDomain{}, errors.New("sender domain transaction service is not configured")
\t}
\ttx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
\tif err != nil {
\t\treturn SenderDomain{}, fmt.Errorf("begin %s: %w", operation, err)
\t}
\tdefer func() { _ = tx.Rollback(ctx) }()
\tvalue, err := fn(s.repository.WithTx(tx))
\tif err != nil {
\t\treturn SenderDomain{}, err
\t}
\tif err := tx.Commit(ctx); err != nil {
\t\treturn SenderDomain{}, fmt.Errorf("commit %s: %w", operation, err)
\t}
\treturn value, nil
}

func (s *Service) createDomain(ctx context.Context, input CreateDomainInput) (SenderDomain, error) {
\treturn s.withRepositoryTx(ctx, "sender domain creation", func(repository *Repository) (SenderDomain, error) {
\t\treturn repository.Create(ctx, input)
\t})
}

func (s *Service) updateVerification(
\tctx context.Context,
\tid, teamID uuid.UUID,
\tstatus string,
\trecords []VerificationRecord,
\tfailureReason *string,
) (SenderDomain, error) {
\treturn s.withRepositoryTx(ctx, "sender domain verification update", func(repository *Repository) (SenderDomain, error) {
\t\treturn repository.UpdateVerification(ctx, id, teamID, status, records, failureReason)
\t})
}

func (s *Service) updateManualHealthCheck(
\tctx context.Context,
\tid, teamID uuid.UUID,
\trecords []VerificationRecord,
\tfailureReason *string,
) (SenderDomain, error) {
\treturn s.withRepositoryTx(ctx, "sender domain manual health update", func(repository *Repository) (SenderDomain, error) {
\t\treturn repository.UpdateManualHealthCheck(ctx, id, teamID, records, failureReason)
\t})
}
'''
if helper_marker not in s:
    raise SystemExit('service helper insertion marker missing')
s = s.replace(helper_marker, '\n' + helpers + helper_marker, 1)
service_path.write_text(s)

reconciliation_path = Path('server/internal/modules/domain/reconciliation.go')
s = reconciliation_path.read_text()
s = s.replace('"github.com/google/uuid"\n', '"github.com/google/uuid"\n\t"github.com/jackc/pgx/v5"\n\t"github.com/jackc/pgx/v5/pgxpool"\n')
s = s.replace('type ReconciliationService struct {\n\trepository *Repository', 'type ReconciliationService struct {\n\tdb         *pgxpool.Pool\n\trepository *Repository')
s = s.replace('func NewReconciliationService(repository *Repository, checker reconciliationChecker, config JobConfig, workerID string) *ReconciliationService {', 'func NewReconciliationService(db *pgxpool.Pool, repository *Repository, checker reconciliationChecker, config JobConfig, workerID string) *ReconciliationService {', 1)
s = s.replace('\t\trepository: repository,\n', '\t\tdb:         db,\n\t\trepository: repository,\n', 1)
s = s.replace('if s == nil || s.repository == nil || s.checker == nil {', 'if s == nil || s.db == nil || s.repository == nil || s.checker == nil {')
s = s.replace('updated, err := s.repository.CompleteReconciliation(ctx, id, s.workerID, result.Status, result.VerificationRecords, nextCheckAt)', 'updated, err := s.completeReconciliation(ctx, id, result.Status, result.VerificationRecords, nextCheckAt)')
insert_marker = '\nfunc (s *ReconciliationService) completeHealthCheck('
helper = '''
func (s *ReconciliationService) completeReconciliation(
\tctx context.Context,
\tid uuid.UUID,
\tstatus string,
\trecords []VerificationRecord,
\tnextCheckAt time.Time,
) (SenderDomain, error) {
\ttx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
\tif err != nil {
\t\treturn SenderDomain{}, fmt.Errorf("begin sender domain reconciliation completion: %w", err)
\t}
\tdefer func() { _ = tx.Rollback(ctx) }()
\tupdated, err := s.repository.WithTx(tx).CompleteReconciliation(ctx, id, s.workerID, status, records, nextCheckAt)
\tif err != nil {
\t\treturn SenderDomain{}, err
\t}
\tif err := tx.Commit(ctx); err != nil {
\t\treturn SenderDomain{}, fmt.Errorf("commit sender domain reconciliation completion: %w", err)
\t}
\treturn updated, nil
}
'''
if insert_marker not in s:
    raise SystemExit('reconciliation helper marker missing')
s = s.replace(insert_marker, '\n' + helper + insert_marker, 1)
reconciliation_path.write_text(s)

jobs_path = Path('server/internal/modules/domain/jobs.go')
s = jobs_path.read_text()
s = s.replace('"github.com/google/uuid"\n', '"github.com/google/uuid"\n\t"github.com/jackc/pgx/v5/pgxpool"\n')
s = s.replace('func NewJob(repository *Repository, checker reconciliationChecker, config JobConfig, workerID string) (*Job, error) {', 'func NewJob(db *pgxpool.Pool, repository *Repository, checker reconciliationChecker, config JobConfig, workerID string) (*Job, error) {', 1)
s = s.replace('if repository == nil || checker == nil {', 'if db == nil || repository == nil || checker == nil {', 1)
s = s.replace('service:    NewReconciliationService(repository, checker, config, workerID),', 'service:    NewReconciliationService(db, repository, checker, config, workerID),', 1)
jobs_path.write_text(s)

for path in [Path('server/internal/registry/server/modules.go'), Path('server/internal/registry/worker/modules.go')]:
    s = path.read_text()
    s = s.replace('domainmodule.NewService(domainRepository, registry.emailClient, dnsVerifier, emailTenantService).WithNotifier(notificationEmailService)', 'domainmodule.NewService(domainRepository, registry.emailClient, dnsVerifier, emailTenantService).WithDatabase(db).WithNotifier(notificationEmailService)')
    s = s.replace('domainmodule.NewService(domainRepository, emailSender, dnsVerifier)', 'domainmodule.NewService(domainRepository, emailSender, dnsVerifier).WithDatabase(db)')
    s = s.replace('domainJob, err := domainmodule.NewJob(\n\t\tdomainRepository,', 'domainJob, err := domainmodule.NewJob(\n\t\tdb,\n\t\tdomainRepository,')
    path.write_text(s)
