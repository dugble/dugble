from pathlib import Path
import re

repository_path = Path('server/internal/modules/webhooks/repository.go')
s = repository_path.read_text()
s = s.replace('\t"github.com/dugble/dugble/server/internal/adapters/postgres"\n', '')
s = s.replace('type Repository struct {\n\tdb      *pgxpool.Pool\n\tqueries *dbsqlc.Queries\n}\n\nfunc NewRepository(db *pgxpool.Pool) *Repository {\n\treturn &Repository{db: db, queries: dbsqlc.New(db)}\n}\n\nfunc (r *Repository) InTransaction(ctx context.Context, operation func(pgx.Tx) error) error {\n\treturn postgres.InTransaction(ctx, r.db, operation)\n}\n', 'type Repository struct {\n\tqueries *dbsqlc.Queries\n}\n\nfunc NewRepository(db *pgxpool.Pool) *Repository {\n\treturn &Repository{queries: dbsqlc.New(db)}\n}\n\nfunc (r *Repository) WithTx(tx pgx.Tx) *Repository {\n\treturn &Repository{queries: r.queries.WithTx(tx)}\n}\n')

update_pattern = re.compile(r'''func \(r \*Repository\) UpdateEndpoint\(ctx context\.Context, id, teamID uuid\.UUID, endpoint validatedEndpoint\) \(Endpoint, error\) \{.*?\n\}\n\nfunc \(r \*Repository\) DisableEndpoint''', re.S)
update_replacement = '''func (r *Repository) UpdateEndpoint(ctx context.Context, id, teamID uuid.UUID, endpoint validatedEndpoint) (Endpoint, error) {
\trow, err := r.queries.UpdateWebhookEndpoint(ctx, dbsqlc.UpdateWebhookEndpointParams{
\t\tID: id, TeamID: teamID, Url: endpoint.URL, Enabled: endpoint.Enabled, SubscribedEvents: endpoint.SubscribedEvents,
\t})
\tif err != nil {
\t\treturn Endpoint{}, fmt.Errorf("update webhook endpoint: %w", err)
\t}
\tif !endpoint.Enabled {
\t\tif _, err := r.queries.CancelWebhookDeliveriesForEndpoint(ctx, dbsqlc.CancelWebhookDeliveriesForEndpointParams{EndpointID: id}); err != nil {
\t\t\treturn Endpoint{}, fmt.Errorf("cancel webhook endpoint deliveries: %w", err)
\t\t}
\t}
\treturn endpointFromSQLC(row), nil
}

func (r *Repository) DisableEndpoint'''
s, count = update_pattern.subn(update_replacement, s, count=1)
if count != 1:
    raise SystemExit('UpdateEndpoint block not found')

disable_pattern = re.compile(r'''func \(r \*Repository\) DisableEndpoint\(ctx context\.Context, id, teamID uuid\.UUID\) \(Endpoint, error\) \{.*?\n\}\n\nfunc \(r \*Repository\) RotateSecret''', re.S)
disable_replacement = '''func (r *Repository) DisableEndpoint(ctx context.Context, id, teamID uuid.UUID) (Endpoint, error) {
\trow, err := r.queries.DisableWebhookEndpoint(ctx, dbsqlc.DisableWebhookEndpointParams{ID: id, TeamID: teamID})
\tif err != nil {
\t\treturn Endpoint{}, fmt.Errorf("disable webhook endpoint: %w", err)
\t}
\tif _, err := r.queries.CancelWebhookDeliveriesForEndpoint(ctx, dbsqlc.CancelWebhookDeliveriesForEndpointParams{EndpointID: id}); err != nil {
\t\treturn Endpoint{}, fmt.Errorf("cancel webhook endpoint deliveries: %w", err)
\t}
\treturn endpointFromSQLC(row), nil
}

func (r *Repository) RotateSecret'''
s, count = disable_pattern.subn(disable_replacement, s, count=1)
if count != 1:
    raise SystemExit('DisableEndpoint block not found')
repository_path.write_text(s)

service_path = Path('server/internal/modules/webhooks/service.go')
s = service_path.read_text()
s = s.replace('\t"errors"\n', '\t"errors"\n\t"fmt"\n')
s = s.replace('\t"github.com/jackc/pgx/v5"\n', '\t"github.com/jackc/pgx/v5"\n\t"github.com/jackc/pgx/v5/pgxpool"\n')
s = s.replace('type Service struct {\n\trepository *Repository', 'type Service struct {\n\tdb         *pgxpool.Pool\n\trepository *Repository')
s = s.replace('func NewService(repository *Repository, emitter *platformwebhook.Emitter) *Service {\n\treturn &Service{repository: repository, emitter: emitter, now: time.Now}\n}\n', 'func NewService(db *pgxpool.Pool, repository *Repository, emitter *platformwebhook.Emitter) *Service {\n\treturn &Service{db: db, repository: repository, emitter: emitter, now: time.Now}\n}\n\nfunc (s *Service) inTransaction(ctx context.Context, operation string, fn func(pgx.Tx) error) error {\n\tif s == nil || s.db == nil {\n\t\treturn errors.New("webhook transaction service is not configured")\n\t}\n\ttx, err := s.db.BeginTx(ctx, pgx.TxOptions{})\n\tif err != nil {\n\t\treturn fmt.Errorf("begin %s: %w", operation, err)\n\t}\n\tdefer func() { _ = tx.Rollback(ctx) }()\n\tif err := fn(tx); err != nil {\n\t\treturn err\n\t}\n\tif err := tx.Commit(ctx); err != nil {\n\t\treturn fmt.Errorf("commit %s: %w", operation, err)\n\t}\n\treturn nil\n}\n')
s = s.replace('endpoint, err := s.repository.UpdateEndpoint(ctx, id, tenantContext.Scope.TeamID, validated)', 'endpoint, err := s.updateEndpoint(ctx, id, tenantContext.Scope.TeamID, validated)')
s = s.replace('endpoint, err := s.repository.DisableEndpoint(ctx, id, tenantContext.Scope.TeamID)', 'endpoint, err := s.disableEndpoint(ctx, id, tenantContext.Scope.TeamID)')
s = s.replace('err = s.repository.InTransaction(ctx, func(tx pgx.Tx) error {', 'err = s.inTransaction(ctx, "webhook test delivery", func(tx pgx.Tx) error {')

helper_marker = '\nfunc requireTenant(ctx context.Context, permission authz.Permission) (authz.Access, error) {'
helpers = '''
func (s *Service) updateEndpoint(ctx context.Context, id, teamID uuid.UUID, endpoint validatedEndpoint) (Endpoint, error) {
\tvar updated Endpoint
\terr := s.inTransaction(ctx, "webhook endpoint update", func(tx pgx.Tx) error {
\t\tvar updateErr error
\t\tupdated, updateErr = s.repository.WithTx(tx).UpdateEndpoint(ctx, id, teamID, endpoint)
\t\treturn updateErr
\t})
\treturn updated, err
}

func (s *Service) disableEndpoint(ctx context.Context, id, teamID uuid.UUID) (Endpoint, error) {
\tvar disabled Endpoint
\terr := s.inTransaction(ctx, "webhook endpoint disable", func(tx pgx.Tx) error {
\t\tvar disableErr error
\t\tdisabled, disableErr = s.repository.WithTx(tx).DisableEndpoint(ctx, id, teamID)
\t\treturn disableErr
\t})
\treturn disabled, err
}
'''
if helper_marker not in s:
    raise SystemExit('service helper marker not found')
s = s.replace(helper_marker, '\n' + helpers + helper_marker, 1)
service_path.write_text(s)

registry_path = Path('server/internal/registry/server/modules.go')
s = registry_path.read_text()
old = 'webhookService := webhooksmodule.NewService(webhookRepository, webhookEmitter)'
new = 'webhookService := webhooksmodule.NewService(db, webhookRepository, webhookEmitter)'
if old not in s:
    raise SystemExit('webhook service registry wiring not found')
s = s.replace(old, new, 1)
registry_path.write_text(s)
