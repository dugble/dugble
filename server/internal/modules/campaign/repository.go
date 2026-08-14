package smscampaign

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	dbsqlc "github.com/dugble/dugble/server/internal/database/sqlc"
	"github.com/dugble/dugble/server/pkg/pgconv"
)

var (
	ErrNotFound         = errors.New("sms campaign not found")
	ErrConflict         = errors.New("sms campaign conflict")
	ErrInvalidReference = errors.New("sms campaign segment or sender does not exist")
)

type Repository struct {
	db      *pgxpool.Pool
	queries *dbsqlc.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db, queries: dbsqlc.New(db)} }

func (r *Repository) Create(ctx context.Context, teamID, segmentID, senderID uuid.UUID, req CreateRequest) (Campaign, error) {
	row, err := r.queries.CreateSMSCampaign(ctx, dbsqlc.CreateSMSCampaignParams{TeamID: teamID, Name: req.Name, SegmentID: segmentID, SenderID: senderID, Body: req.Body, RateLimitPerSecond: req.RateLimitPerSecond, DailySendLimit: req.DailySendLimit})
	return mapCampaign(row, err)
}
func (r *Repository) List(ctx context.Context, teamID uuid.UUID, req ListRequest) ([]Campaign, error) {
	rows, err := r.queries.ListSMSCampaigns(ctx, dbsqlc.ListSMSCampaignsParams{TeamID: teamID, PageLimit: req.Limit, PageOffset: req.Offset})
	if err != nil {
		return nil, fmt.Errorf("list SMS campaigns: %w", err)
	}
	result := make([]Campaign, 0, len(rows))
	for _, row := range rows {
		value, mapErr := mapCampaign(row, nil)
		if mapErr != nil {
			return nil, mapErr
		}
		result = append(result, value)
	}
	return result, nil
}
func (r *Repository) Get(ctx context.Context, teamID, id uuid.UUID) (Campaign, error) {
	row, err := r.queries.GetSMSCampaign(ctx, dbsqlc.GetSMSCampaignParams{ID: id, TeamID: teamID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Campaign{}, ErrNotFound
	}
	return mapCampaign(row, err)
}
func (r *Repository) Update(ctx context.Context, teamID, id, segmentID, senderID uuid.UUID, req UpdateRequest, value Campaign) (Campaign, error) {
	return mapCampaign(r.queries.UpdateSMSCampaignDraft(ctx, dbsqlc.UpdateSMSCampaignDraftParams{Name: value.Name, SegmentID: segmentID, SenderID: senderID, Body: value.Body, RateLimitPerSecond: value.RateLimitPerSecond, DailySendLimit: value.DailySendLimit, ID: id, TeamID: teamID, Revision: req.Revision}))
}
func (r *Repository) Delete(ctx context.Context, teamID, id uuid.UUID) (Campaign, error) {
	return mapCampaign(r.queries.DeleteSMSCampaign(ctx, dbsqlc.DeleteSMSCampaignParams{ID: id, TeamID: teamID}))
}
func (r *Repository) Duplicate(ctx context.Context, teamID, id uuid.UUID, name string) (Campaign, error) {
	return mapCampaign(r.queries.DuplicateSMSCampaign(ctx, dbsqlc.DuplicateSMSCampaignParams{Name: name, SourceID: id, TeamID: teamID}))
}
func (r *Repository) HasApprovedSender(ctx context.Context, teamID, id uuid.UUID) (bool, error) {
	return r.queries.IsApprovedSMSCampaignSender(ctx, dbsqlc.IsApprovedSMSCampaignSenderParams{ID: id, TeamID: teamID})
}
func (r *Repository) Activate(ctx context.Context, teamID, id uuid.UUID, scheduledAt *time.Time) (Campaign, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Campaign{}, fmt.Errorf("begin SMS campaign activation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := r.queries.WithTx(tx)
	row, err := q.ActivateSMSCampaign(ctx, dbsqlc.ActivateSMSCampaignParams{ScheduledAt: pgconv.NullableTimestamptz(scheduledAt), ID: id, TeamID: teamID})
	value, err := mapCampaign(row, err)
	if err != nil {
		return Campaign{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Campaign{}, fmt.Errorf("commit SMS campaign activation: %w", err)
	}
	return value, nil
}
func (r *Repository) Cancel(ctx context.Context, teamID, id uuid.UUID) (Campaign, error) {
	return mapCampaign(r.queries.CancelSMSCampaign(ctx, dbsqlc.CancelSMSCampaignParams{ID: id, TeamID: teamID}))
}
func (r *Repository) ListRecipients(ctx context.Context, teamID, id uuid.UUID, req ListRequest) ([]Recipient, error) {
	rows, err := r.queries.ListSMSCampaignRecipients(ctx, dbsqlc.ListSMSCampaignRecipientsParams{CampaignID: id, TeamID: teamID, PageLimit: req.Limit, PageOffset: req.Offset})
	if err != nil {
		return nil, fmt.Errorf("list SMS campaign recipients: %w", err)
	}
	result := make([]Recipient, 0, len(rows))
	for _, row := range rows {
		var snapshot map[string]any
		if err := json.Unmarshal(row.ContactSnapshot, &snapshot); err != nil {
			return nil, fmt.Errorf("decode recipient snapshot: %w", err)
		}
		result = append(result, Recipient{ID: row.ID.String(), CampaignID: row.CampaignID.String(), ContactID: uuidPtrString(row.ContactID), Phone: row.Phone, PhoneCountry: row.PhoneCountry, ContactSnapshot: snapshot, Status: row.Status, ExclusionReason: row.ExclusionReason, SMSMessageID: uuidPtrString(row.SmsMessageID), CreatedAt: pgconv.TimestamptzToTime(row.CreatedAt), QueuedAt: pgconv.TimestamptzToTimePtr(row.QueuedAt), RenderedBody: row.RenderedBody, AttemptCount: row.AttemptCount, FailureCode: row.FailureCode, FailureMessage: row.FailureMessage, Encoding: row.Encoding, EstimatedSegments: row.EstimatedSegments, EstimatedUnitCostUnits: row.EstimatedUnitCostUnits, EstimatedCostUnits: row.EstimatedCostUnits, ActualSegments: row.ActualSegments, ActualChargeUnits: row.ActualChargeUnits})
	}
	return result, nil
}

func mapCampaign(row dbsqlc.SmsCampaign, err error) (Campaign, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return Campaign{}, ErrConflict
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return Campaign{}, ErrInvalidReference
	}
	if err != nil {
		return Campaign{}, fmt.Errorf("SMS campaign query: %w", err)
	}
	return Campaign{ID: row.ID.String(), TeamID: row.TeamID.String(), Name: row.Name, Status: row.Status, SegmentID: row.SegmentID.String(), SenderID: row.SenderID.String(), Body: row.Body, ScheduledAt: pgconv.TimestamptzToTimePtr(row.ScheduledAt), QueuedAt: pgconv.TimestamptzToTimePtr(row.QueuedAt), CanceledAt: pgconv.TimestamptzToTimePtr(row.CanceledAt), MaterializedAt: pgconv.TimestamptzToTimePtr(row.MaterializedAt), SentAt: pgconv.TimestamptzToTimePtr(row.SentAt), AudienceCount: row.AudienceCount, EligibleCount: row.EligibleCount, ExcludedCount: row.ExcludedCount, FailedCount: row.FailedCount, EstimatedSegments: row.EstimatedSegments, EstimatedCostUnits: row.EstimatedCostUnits, EstimatedBillableCostUnits: row.EstimatedBillableCostUnits, PreflightAllowanceSegments: row.PreflightAllowanceSegments, ActualSegments: row.ActualSegments, ActualChargeUnits: row.ActualChargeUnits, Currency: row.Currency, PreflightBalanceUnits: row.PreflightBalanceUnits, PreflightAt: pgconv.TimestamptzToTimePtr(row.PreflightAt), RateLimitPerSecond: row.RateLimitPerSecond, DailySendLimit: row.DailySendLimit, Revision: row.Revision, CreatedAt: pgconv.TimestamptzToTime(row.CreatedAt), UpdatedAt: pgconv.TimestamptzToTime(row.UpdatedAt)}, nil
}

func (r *Repository) RecordOptOut(ctx context.Context, teamID uuid.UUID, req RecordOptOutRequest) (ConsentEvent, error) {
	row, err := r.queries.RecordSMSOptOut(ctx, dbsqlc.RecordSMSOptOutParams{TeamID: teamID, Phone: req.Phone, Source: req.Source, SourceID: req.SourceID})
	if err != nil {
		return ConsentEvent{}, fmt.Errorf("record SMS opt-out: %w", err)
	}
	return ConsentEvent{ID: row.ID.String(), ContactID: uuidPtrString(row.ContactID), Phone: row.Phone, Action: row.Action, Source: row.Source, SourceID: row.SourceID, RecordedAt: pgconv.TimestamptzToTime(row.RecordedAt)}, nil
}

func (r *Repository) GetExclusionSummary(ctx context.Context, teamID, campaignID uuid.UUID) (ExclusionSummary, error) {
	if _, err := r.Get(ctx, teamID, campaignID); err != nil {
		return ExclusionSummary{}, err
	}
	rows, err := r.queries.GetSMSCampaignExclusionSummary(ctx, dbsqlc.GetSMSCampaignExclusionSummaryParams{CampaignID: campaignID, TeamID: teamID})
	if err != nil {
		return ExclusionSummary{}, fmt.Errorf("get SMS campaign exclusion summary: %w", err)
	}
	result := ExclusionSummary{CampaignID: campaignID.String(), Reasons: map[string]int64{}}
	for _, row := range rows {
		reason := "unknown"
		if row.ExclusionReason != nil {
			reason = *row.ExclusionReason
		}
		result.Reasons[reason] = row.Total
		result.Total += row.Total
	}
	return result, nil
}

func (r *Repository) GetAnalytics(ctx context.Context, teamID, campaignID uuid.UUID) (Analytics, error) {
	row, err := r.queries.GetSMSCampaignAnalytics(ctx, dbsqlc.GetSMSCampaignAnalyticsParams{CampaignID: campaignID, TeamID: teamID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Analytics{}, ErrNotFound
	}
	if err != nil {
		return Analytics{}, fmt.Errorf("get SMS campaign analytics: %w", err)
	}
	return Analytics{CampaignID: row.CampaignID.String(), Audience: row.AudienceCount, Eligible: row.EligibleCount, Excluded: row.ExcludedCount, Queued: row.QueuedCount, Failed: row.FailedCount, Sent: row.SentCount, Delivered: row.DeliveredCount, DeliveryFailed: row.DeliveryFailedCount, EstimatedSegments: row.EstimatedSegments, EstimatedCostUnits: row.EstimatedCostUnits, EstimatedBillableCostUnits: row.EstimatedBillableCostUnits, ActualSegments: row.ActualSegments, ActualChargeUnits: row.ActualChargeUnits, Currency: row.Currency}, nil
}
func uuidPtrString(value *uuid.UUID) *string {
	if value == nil {
		return nil
	}
	result := value.String()
	return &result
}

func (r *Repository) QueueNextDue(ctx context.Context) (Campaign, bool, error) {
	row, err := r.queries.QueueNextDueSMSCampaign(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return Campaign{}, false, nil
	}
	value, err := mapCampaign(row, err)
	return value, err == nil, err
}

func (r *Repository) MaterializeNext(ctx context.Context) (Campaign, bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Campaign{}, false, fmt.Errorf("begin SMS campaign materialization: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := r.queries.WithTx(tx)
	row, err := q.ClaimNextSMSCampaignForMaterialization(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return Campaign{}, false, nil
	}
	if err != nil {
		return Campaign{}, false, fmt.Errorf("claim SMS campaign for materialization: %w", err)
	}
	if _, err = q.MaterializeClaimedSMSCampaignRecipients(ctx, dbsqlc.MaterializeClaimedSMSCampaignRecipientsParams{CampaignID: row.ID, TeamID: row.TeamID}); err != nil {
		return Campaign{}, false, fmt.Errorf("materialize SMS campaign recipients: %w", err)
	}
	row, err = q.FinishSMSCampaignMaterialization(ctx, dbsqlc.FinishSMSCampaignMaterializationParams{ID: row.ID, TeamID: row.TeamID})
	value, err := mapCampaign(row, err)
	if err != nil {
		return Campaign{}, false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Campaign{}, false, fmt.Errorf("commit SMS campaign materialization: %w", err)
	}
	return value, true, nil
}

func (r *Repository) BeginFanoutTx(ctx context.Context) (pgx.Tx, error) { return r.db.Begin(ctx) }

func (r *Repository) ClaimNextRecipientTx(ctx context.Context, tx pgx.Tx) (FanoutRecipient, bool, error) {
	row, err := r.queries.WithTx(tx).ClaimNextSMSCampaignRecipient(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return FanoutRecipient{}, false, nil
	}
	if err != nil {
		return FanoutRecipient{}, false, fmt.Errorf("claim SMS campaign recipient: %w", err)
	}
	if row.Phone == nil || row.PhoneCountry == nil {
		return FanoutRecipient{}, false, errors.New("eligible SMS campaign recipient has no phone identity")
	}
	var snapshot map[string]any
	if err := json.Unmarshal(row.ContactSnapshot, &snapshot); err != nil {
		return FanoutRecipient{}, false, fmt.Errorf("decode SMS campaign recipient snapshot: %w", err)
	}
	rendered := ""
	if row.RenderedBody != nil {
		rendered = *row.RenderedBody
	}
	return FanoutRecipient{ID: row.ID, TeamID: row.TeamID, CampaignID: row.CampaignID, ContactID: row.ContactID, Phone: *row.Phone, PhoneCountry: *row.PhoneCountry, ContactSnapshot: snapshot, CampaignBody: row.CampaignBody, RenderedBody: rendered, SenderID: row.SenderID, SenderName: row.SenderName, AttemptCount: row.AttemptCount}, true, nil
}

func (r *Repository) RecheckRecipientTx(ctx context.Context, tx pgx.Tx, recipient FanoutRecipient) (string, bool, error) {
	reason, err := r.queries.WithTx(tx).RecheckSMSCampaignRecipientEligibility(ctx, dbsqlc.RecheckSMSCampaignRecipientEligibilityParams{RecipientID: recipient.ID, TeamID: recipient.TeamID})
	if err != nil {
		return "", false, fmt.Errorf("recheck SMS campaign recipient: %w", err)
	}
	return reason, reason != "", nil
}

func (r *Repository) SetRecipientQueuedTx(ctx context.Context, tx pgx.Tx, recipient FanoutRecipient, messageID uuid.UUID, body string, segments int32, chargeUnits int64) error {
	return r.queries.WithTx(tx).SetSMSCampaignRecipientQueued(ctx, dbsqlc.SetSMSCampaignRecipientQueuedParams{SmsMessageID: &messageID, RenderedBody: &body, ActualSegments: &segments, ActualChargeUnits: &chargeUnits, ID: recipient.ID, TeamID: recipient.TeamID})
}

func (r *Repository) ClaimNextEstimateTx(ctx context.Context, tx pgx.Tx) (FanoutRecipient, bool, error) {
	row, err := r.queries.WithTx(tx).ClaimNextSMSCampaignRecipientForEstimate(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return FanoutRecipient{}, false, nil
	}
	if err != nil {
		return FanoutRecipient{}, false, fmt.Errorf("claim SMS campaign recipient estimate: %w", err)
	}
	if row.Phone == nil || row.PhoneCountry == nil {
		return FanoutRecipient{}, false, errors.New("estimated SMS recipient has no phone identity")
	}
	var snapshot map[string]any
	if err = json.Unmarshal(row.ContactSnapshot, &snapshot); err != nil {
		return FanoutRecipient{}, false, fmt.Errorf("decode estimated recipient snapshot: %w", err)
	}
	return FanoutRecipient{ID: row.ID, TeamID: row.TeamID, CampaignID: row.CampaignID, ContactID: row.ContactID, Phone: *row.Phone, PhoneCountry: *row.PhoneCountry, ContactSnapshot: snapshot, CampaignBody: row.CampaignBody, SenderID: row.SenderID, SenderName: row.SenderName}, true, nil
}

func (r *Repository) SetRecipientEstimateTx(ctx context.Context, tx pgx.Tx, recipient FanoutRecipient, body, encoding string, segments int32) error {
	q := r.queries.WithTx(tx)
	cost, err := q.EstimateSMSCampaignRecipientCost(ctx, dbsqlc.EstimateSMSCampaignRecipientCostParams{Segments: int64(segments), DestinationCountry: recipient.PhoneCountry, TeamID: recipient.TeamID})
	if err != nil {
		return fmt.Errorf("price SMS campaign recipient: %w", err)
	}
	return q.SetSMSCampaignRecipientEstimate(ctx, dbsqlc.SetSMSCampaignRecipientEstimateParams{RenderedBody: &body, Encoding: &encoding, EstimatedSegments: &segments, EstimatedUnitCostUnits: &cost.UnitCostUnits, EstimatedCostUnits: &cost.CostUnits, ID: recipient.ID, TeamID: recipient.TeamID})
}

func (r *Repository) FailRecipientEstimateTx(ctx context.Context, tx pgx.Tx, recipient FanoutRecipient, code, message string) error {
	return r.queries.WithTx(tx).FailSMSCampaignRecipientEstimate(ctx, dbsqlc.FailSMSCampaignRecipientEstimateParams{FailureCode: &code, FailureMessage: &message, ID: recipient.ID, TeamID: recipient.TeamID})
}

func (r *Repository) FinalizeCostPreflightTx(ctx context.Context, tx pgx.Tx, recipient FanoutRecipient) (Campaign, bool, error) {
	row, err := r.queries.WithTx(tx).FinalizeSMSCampaignCostPreflight(ctx, dbsqlc.FinalizeSMSCampaignCostPreflightParams{ID: recipient.CampaignID, TeamID: recipient.TeamID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Campaign{}, false, nil
	}
	value, err := mapCampaign(row, err)
	return value, err == nil, err
}

func (r *Repository) GetCostEstimate(ctx context.Context, teamID, id uuid.UUID) (CostEstimate, error) {
	row, err := r.queries.GetSMSCampaignCostEstimate(ctx, dbsqlc.GetSMSCampaignCostEstimateParams{ID: id, TeamID: teamID})
	if errors.Is(err, pgx.ErrNoRows) {
		return CostEstimate{}, ErrNotFound
	}
	if err != nil {
		return CostEstimate{}, fmt.Errorf("get SMS campaign cost estimate: %w", err)
	}
	return CostEstimate{CampaignID: row.CampaignID.String(), Currency: row.Currency, Recipients: row.Recipients, EstimatedSegments: row.EstimatedSegments, EstimatedCostUnits: row.EstimatedCostUnits, EstimatedBillableCostUnits: row.EstimatedBillableCostUnits, PreflightAllowanceSegments: row.PreflightAllowanceSegments, MinimumRecipientCostUnits: row.MinimumRecipientCostUnits, MaximumRecipientCostUnits: row.MaximumRecipientCostUnits, ActualSegments: row.ActualSegments, ActualChargeUnits: row.ActualChargeUnits, PreflightBalanceUnits: row.PreflightBalanceUnits, PreflightAt: pgconv.TimestamptzToTimePtr(row.PreflightAt)}, nil
}

func (r *Repository) FailRecipientTx(ctx context.Context, tx pgx.Tx, recipient FanoutRecipient, code, message string) error {
	return r.queries.WithTx(tx).FailSMSCampaignRecipient(ctx, dbsqlc.FailSMSCampaignRecipientParams{FailureCode: &code, FailureMessage: &message, ID: recipient.ID, TeamID: recipient.TeamID})
}

func (r *Repository) FinalizeFanoutTx(ctx context.Context, tx pgx.Tx, teamID, campaignID uuid.UUID) (Campaign, bool, error) {
	row, err := r.queries.WithTx(tx).FinalizeSMSCampaignFanout(ctx, dbsqlc.FinalizeSMSCampaignFanoutParams{ID: campaignID, TeamID: teamID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Campaign{}, false, nil
	}
	value, err := mapCampaign(row, err)
	return value, err == nil, err
}
