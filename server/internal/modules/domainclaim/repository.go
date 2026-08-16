package domainclaim

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	dbsqlc "github.com/dugble/dugble/server/internal/database/sqlc"
	platformemail "github.com/dugble/dugble/server/internal/providers/aws/ses"
	"github.com/dugble/dugble/server/pkg/pgconv"
)

type Repository struct {
	queries *dbsqlc.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{queries: dbsqlc.New(db)}
}

func (r *Repository) WithTx(tx pgx.Tx) *Repository {
	return &Repository{queries: r.queries.WithTx(tx)}
}

func (r *Repository) ActiveSourceByName(ctx context.Context, name string) (sourceDomain, error) {
	row, err := r.queries.GetActiveDomainForClaimByName(ctx, dbsqlc.GetActiveDomainForClaimByNameParams{Name: name})
	if err != nil {
		return sourceDomain{}, err
	}
	return sourceFromSQLC(row), nil
}

func (r *Repository) Create(ctx context.Context, targetTeamID, createdBy uuid.UUID, source sourceDomain, region string, cfg configuration, now time.Time) (Claim, error) {
	sourceID, err := uuid.Parse(source.ID)
	if err != nil {
		return Claim{}, fmt.Errorf("parse source domain id: %w", err)
	}
	sourceTeamID, err := uuid.Parse(source.TeamID)
	if err != nil {
		return Claim{}, fmt.Errorf("parse source team id: %w", err)
	}
	row, err := r.queries.CreateDomainClaim(ctx, dbsqlc.CreateDomainClaimParams{
		TargetDomainID:   uuid.New(),
		SourceDomainID:   ptrUUID(sourceID),
		NormalizedName:   source.Name,
		SourceTeamID:     sourceTeamID,
		TargetTeamID:     targetTeamID,
		ProviderRegion:   region,
		CustomReturnPath: cfg.CustomReturnPath,
		TlsMode:          cfg.TLS,
		RecordName:       "_dugble-claim." + source.Name,
		RecordValue:      "dgb_claim_" + strings.ReplaceAll(uuid.NewString(), "-", ""),
		RecordTtl:        "Auto",
		ExpiresAt:        pgconv.TimestamptzFromTime(now.Add(DefaultClaimLifetime)),
		CreatedBy:        ptrUUID(createdBy),
	})
	if isUniqueViolation(err) {
		return Claim{}, ErrClaimAlreadyExists
	}
	if err != nil {
		return Claim{}, fmt.Errorf("create domain claim: %w", err)
	}
	return claimFromSQLC(row), nil
}

func (r *Repository) Get(ctx context.Context, targetDomainID, targetTeamID uuid.UUID) (Claim, error) {
	row, err := r.queries.GetDomainClaim(ctx, dbsqlc.GetDomainClaimParams{TargetDomainID: targetDomainID, TargetTeamID: targetTeamID})
	if err != nil {
		return Claim{}, err
	}
	return claimFromSQLC(row), nil
}

func (r *Repository) RequestVerification(ctx context.Context, targetDomainID, targetTeamID uuid.UUID) (Claim, error) {
	row, err := r.queries.RequestDomainClaimVerification(ctx, dbsqlc.RequestDomainClaimVerificationParams{TargetDomainID: targetDomainID, TargetTeamID: targetTeamID})
	if err != nil {
		return Claim{}, err
	}
	return claimFromSQLC(row), nil
}

func (r *Repository) Cancel(ctx context.Context, targetDomainID, targetTeamID uuid.UUID) (Claim, error) {
	row, err := r.queries.CancelDomainClaim(ctx, dbsqlc.CancelDomainClaimParams{TargetDomainID: targetDomainID, TargetTeamID: targetTeamID})
	if err != nil {
		return Claim{}, err
	}
	return claimFromSQLC(row), nil
}

func (r *Repository) ClaimPending(ctx context.Context, workerID string, limit int32, staleBefore time.Time) ([]ReconciliationClaim, error) {
	rows, err := r.queries.ClaimPendingDomainClaims(ctx, dbsqlc.ClaimPendingDomainClaimsParams{
		StaleBefore: pgconv.TimestamptzFromTime(staleBefore), ClaimLimit: limit, WorkerID: strings.TrimSpace(workerID),
	})
	if err != nil {
		return nil, fmt.Errorf("claim pending domain claims: %w", err)
	}
	claims := make([]ReconciliationClaim, 0, len(rows))
	for _, row := range rows {
		claims = append(claims, ReconciliationClaim{Claim: claimFromSQLC(dbsqlc.DomainClaim(row))})
	}
	return claims, nil
}

func (r *Repository) Release(ctx context.Context, id uuid.UUID, workerID string) (Claim, error) {
	row, err := r.queries.ReleaseDomainClaim(ctx, dbsqlc.ReleaseDomainClaimParams{ID: id, WorkerID: strings.TrimSpace(workerID)})
	if err != nil {
		return Claim{}, err
	}
	return claimFromSQLC(row), nil
}

func (r *Repository) MarkVerified(ctx context.Context, id uuid.UUID, workerID string) (Claim, error) {
	row, err := r.queries.MarkDomainClaimVerified(ctx, dbsqlc.MarkDomainClaimVerifiedParams{ID: id, WorkerID: strings.TrimSpace(workerID)})
	if err != nil {
		return Claim{}, err
	}
	return claimFromSQLC(row), nil
}

func (r *Repository) MarkBlocked(ctx context.Context, id uuid.UUID, workerID, reason string) (Claim, error) {
	row, err := r.queries.MarkDomainClaimBlocked(ctx, dbsqlc.MarkDomainClaimBlockedParams{ID: id, WorkerID: strings.TrimSpace(workerID), BlockedReason: &reason})
	if err != nil {
		return Claim{}, err
	}
	return claimFromSQLC(row), nil
}

func (r *Repository) MarkFailed(ctx context.Context, id uuid.UUID, workerID string, cause error) (Claim, error) {
	reason := "domain claim reconciliation failed"
	if cause != nil {
		reason = cause.Error()
	}
	row, err := r.queries.MarkDomainClaimFailed(ctx, dbsqlc.MarkDomainClaimFailedParams{ID: id, WorkerID: strings.TrimSpace(workerID), FailureReason: &reason})
	if err != nil {
		return Claim{}, err
	}
	return claimFromSQLC(row), nil
}

func (r *Repository) Source(ctx context.Context, claim Claim) (sourceDomain, error) {
	if claim.SourceDomainID == nil {
		return sourceDomain{}, pgx.ErrNoRows
	}
	id, err := uuid.Parse(*claim.SourceDomainID)
	if err != nil {
		return sourceDomain{}, err
	}
	row, err := r.queries.GetDomainForClaimByID(ctx, dbsqlc.GetDomainForClaimByIDParams{ID: id})
	if err != nil {
		return sourceDomain{}, err
	}
	return sourceFromSQLC(row), nil
}

func (r *Repository) HasPendingScheduledEmails(ctx context.Context, sourceDomainID uuid.UUID) (bool, error) {
	return r.queries.DomainHasPendingScheduledEmails(ctx, dbsqlc.DomainHasPendingScheduledEmailsParams{DomainID: &sourceDomainID})
}

func (r *Repository) HasRecentOwnerActivity(ctx context.Context, sourceDomainID uuid.UUID, since time.Time) (bool, error) {
	return r.queries.DomainHasRecentOwnerActivity(ctx, dbsqlc.DomainHasRecentOwnerActivityParams{
		DomainID: &sourceDomainID,
		Since:    pgconv.TimestamptzFromTime(since),
	})
}

func (r *Repository) FreezeSource(ctx context.Context, source sourceDomain) error {
	id, err := uuid.Parse(source.ID)
	if err != nil {
		return err
	}
	teamID, err := uuid.Parse(source.TeamID)
	if err != nil {
		return err
	}
	_, err = r.queries.ArchiveClaimSourceDomain(ctx, dbsqlc.ArchiveClaimSourceDomainParams{ID: id, TeamID: teamID})
	if err != nil {
		return fmt.Errorf("archive source sender domain: %w", err)
	}
	return nil
}

func (r *Repository) CompleteTransfer(ctx context.Context, claim Claim, workerID string, records []VerificationRecord) (Claim, error) {
	claimID, err := uuid.Parse(claim.ID)
	if err != nil {
		return Claim{}, err
	}
	targetDomainID, err := uuid.Parse(claim.DomainID)
	if err != nil {
		return Claim{}, err
	}
	targetTeamID, err := uuid.Parse(claim.TargetTeamID)
	if err != nil {
		return Claim{}, err
	}
	locked, err := r.queries.GetDomainClaimByID(ctx, dbsqlc.GetDomainClaimByIDParams{ID: claimID})
	if err != nil {
		return Claim{}, fmt.Errorf("load domain claim for completion: %w", err)
	}
	if locked.Status != StatusVerified {
		return Claim{}, ErrClaimNotReady
	}
	created, err := r.queries.CreateClaimTargetDomain(ctx, dbsqlc.CreateClaimTargetDomainParams{
		ID: targetDomainID, TeamID: targetTeamID, Name: claim.Name,
		Provider: "aws_ses", ProviderAccount: "default", ProviderRegion: claim.Region,
		TlsMode: claim.TLS, CustomReturnPath: claim.CustomReturnPath, CreatedBy: locked.CreatedBy,
	})
	if err != nil {
		return Claim{}, fmt.Errorf("create claimed sender domain: %w", err)
	}
	if err := replaceDNSRecords(ctx, r.queries, created.ID, records); err != nil {
		return Claim{}, err
	}
	completed, err := r.queries.MarkDomainClaimCompleted(ctx, dbsqlc.MarkDomainClaimCompletedParams{ID: claimID, WorkerID: strings.TrimSpace(workerID)})
	if err != nil {
		return Claim{}, fmt.Errorf("mark domain claim completed: %w", err)
	}
	return claimFromSQLC(completed), nil
}

func replaceDNSRecords(ctx context.Context, queries *dbsqlc.Queries, domainID uuid.UUID, records []VerificationRecord) error {
	if err := queries.DeleteCurrentClaimTargetDNSRecords(ctx, dbsqlc.DeleteCurrentClaimTargetDNSRecordsParams{DomainID: domainID}); err != nil {
		return fmt.Errorf("delete current sender domain DNS records: %w", err)
	}
	for _, record := range records {
		var priority *int32
		if record.Priority != nil {
			value := int32(*record.Priority)
			priority = &value
		}
		if _, err := queries.CreateClaimTargetDNSRecord(ctx, dbsqlc.CreateClaimTargetDNSRecordParams{
			DomainID: domainID, Purpose: verificationRecordPurpose(record), Record: record.Record,
			Name: record.Name, Type: record.Type, Value: record.Value, Ttl: record.TTL,
			Priority: priority, Status: record.Status,
		}); err != nil {
			return fmt.Errorf("create claimed sender domain DNS record: %w", err)
		}
	}
	return nil
}

func verificationRecordPurpose(record VerificationRecord) string {
	switch strings.ToUpper(strings.TrimSpace(record.Record)) {
	case platformemail.RecordDKIM:
		return "dkim"
	case platformemail.RecordSPF:
		if strings.EqualFold(strings.TrimSpace(record.Type), platformemail.RecordTypeMX) {
			return "mail_from"
		}
		return "spf"
	default:
		value := strings.ToLower(strings.TrimSpace(record.Record))
		if value == "" {
			return "spf"
		}
		return value
	}
}

func sourceFromSQLC(row dbsqlc.Domain) sourceDomain {
	var createdBy *string
	if row.CreatedBy != nil {
		value := row.CreatedBy.String()
		createdBy = &value
	}
	return sourceDomain{
		ID: row.ID.String(), TeamID: row.TeamID.String(), Name: row.NormalizedName,
		Provider: row.Provider, ProviderAccount: row.ProviderAccount, ProviderRegion: row.ProviderRegion,
		CustomReturnPath: row.CustomReturnPath, Status: row.Status, CreatedBy: createdBy,
		VerifiedAt: pgconv.TimestamptzToTimePtr(row.VerifiedAt), CreatedAt: row.CreatedAt.Time,
	}
}

func claimFromSQLC(row dbsqlc.DomainClaim) Claim {
	var sourceDomainID *string
	if row.SourceDomainID != nil {
		value := row.SourceDomainID.String()
		sourceDomainID = &value
	}
	return Claim{
		ID: row.ID.String(), DomainID: row.TargetDomainID.String(), Name: row.NormalizedName,
		Status: row.Status, SourceDomainID: sourceDomainID, SourceTeamID: row.SourceTeamID.String(),
		TargetTeamID: row.TargetTeamID.String(), Region: row.ProviderRegion,
		CustomReturnPath: row.CustomReturnPath, TLS: row.TlsMode,
		VerificationRecord: VerificationRecord{Record: "claim", Name: row.RecordName, Value: row.RecordValue, Type: platformemail.RecordTypeTXT, Status: claimRecordStatus(row.Status), TTL: row.RecordTtl},
		BlockedReason:      row.BlockedReason, FailureReason: row.FailureReason,
		VerificationRequestedAt: pgconv.TimestamptzToTimePtr(row.VerificationRequestedAt),
		VerifiedAt:              pgconv.TimestamptzToTimePtr(row.VerifiedAt), CompletedAt: pgconv.TimestamptzToTimePtr(row.CompletedAt),
		ExpiresAt: row.ExpiresAt.Time, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
	}
}

func claimRecordStatus(status string) string {
	if status == StatusVerified || status == StatusCompleted {
		return platformemail.RecordStatusVerified
	}
	if status == StatusFailed || status == StatusExpired || status == StatusCanceled {
		return platformemail.RecordStatusFailed
	}
	return platformemail.RecordStatusPending
}

func ptrUUID(value uuid.UUID) *uuid.UUID { return &value }

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
