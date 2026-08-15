package senderid

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	dbsqlc "github.com/dugble/dugble/server/internal/database/sqlc"
	"github.com/dugble/dugble/server/internal/platform/systemmail"
	"github.com/dugble/dugble/server/pkg/pgconv"
)

type Repository struct {
	queries *dbsqlc.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{queries: dbsqlc.New(db)}
}

func (r *Repository) Create(
	ctx context.Context,
	teamID uuid.UUID,
	name string,
	countryCode string,
	purpose string,
	provider *string,
	createdBy *uuid.UUID,
) (SenderID, error) {
	if r == nil || r.queries == nil {
		return SenderID{}, errors.New("sender id repository is not configured")
	}

	normalizedName := strings.TrimSpace(name)
	countryCode = strings.ToUpper(strings.TrimSpace(countryCode))
	var normalizedProvider *string
	if provider != nil {
		value := strings.ToLower(strings.TrimSpace(*provider))
		if value != "" {
			normalizedProvider = &value
		}
	}

	row, err := r.queries.CreateSenderID(ctx, dbsqlc.CreateSenderIDParams{
		TeamID:      teamID,
		Name:        normalizedName,
		CountryCode: countryCode,
		Purpose:     purpose,
		Provider:    normalizedProvider,
		CreatedBy:   createdBy,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return SenderID{}, ErrSenderIDAlreadyExists
		}
		return SenderID{}, fmt.Errorf("create sender id: %w", err)
	}
	return senderIDFromRow(row), nil
}

func (r *Repository) List(ctx context.Context, teamID uuid.UUID) ([]SenderID, error) {
	if r == nil || r.queries == nil {
		return nil, errors.New("sender id repository is not configured")
	}
	rows, err := r.queries.ListSenderIDs(ctx, dbsqlc.ListSenderIDsParams{TeamID: teamID})
	if err != nil {
		return nil, fmt.Errorf("list sender ids: %w", err)
	}
	senders := make([]SenderID, 0, len(rows))
	for _, row := range rows {
		senders = append(senders, senderIDFromRow(row))
	}
	return senders, nil
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID, teamID uuid.UUID) (SenderID, error) {
	if r == nil || r.queries == nil {
		return SenderID{}, errors.New("sender id repository is not configured")
	}
	row, err := r.queries.GetSenderID(ctx, dbsqlc.GetSenderIDParams{ID: id, TeamID: teamID})
	if err != nil {
		return SenderID{}, fmt.Errorf("get sender id: %w", err)
	}
	return senderIDFromRow(row), nil
}

func (r *Repository) Deactivate(ctx context.Context, id uuid.UUID, teamID uuid.UUID) (SenderID, error) {
	if r == nil || r.queries == nil {
		return SenderID{}, errors.New("sender id repository is not configured")
	}
	row, err := r.queries.DeactivateSenderID(ctx, dbsqlc.DeactivateSenderIDParams{ID: id, TeamID: teamID})
	if err != nil {
		return SenderID{}, fmt.Errorf("deactivate sender id: %w", err)
	}
	return senderIDFromRow(row), nil
}

func (r *Repository) ClaimPendingRegistrations(ctx context.Context, workerID, providerID string, limit int32, staleBefore time.Time) ([]RegistrationClaim, error) {
	if r == nil || r.queries == nil {
		return nil, errors.New("sender id repository is not configured")
	}
	workerID = strings.TrimSpace(workerID)
	providerID = strings.ToLower(strings.TrimSpace(providerID))
	if workerID == "" || providerID == "" {
		return nil, errors.New("sender ID reconciliation requires worker and provider IDs")
	}
	if limit <= 0 {
		limit = 25
	}
	rows, err := r.queries.ClaimPendingSenderIDRegistrations(ctx, dbsqlc.ClaimPendingSenderIDRegistrationsParams{
		ProviderID:  providerID,
		WorkerID:    workerID,
		ClaimLimit:  limit,
		StaleBefore: pgconv.TimestamptzFromTime(staleBefore),
	})
	if err != nil {
		return nil, fmt.Errorf("claim pending Sender ID registrations: %w", err)
	}
	claims := make([]RegistrationClaim, 0, len(rows))
	for _, row := range rows {
		claim := RegistrationClaim{
			ID:             row.ID,
			TeamID:         row.TeamID,
			Name:           row.Name,
			CountryCode:    row.CountryCode,
			Provider:       row.Provider,
			ProviderStatus: row.ProviderStatus,
			Attempt:        row.ReconciliationAttempts,
		}
		if row.SubmittedAt.Valid {
			value := row.SubmittedAt.Time
			claim.ProviderSubmittedAt = &value
		}
		claims = append(claims, claim)
	}
	return claims, nil
}

func (r *Repository) CompleteSubmission(ctx context.Context, id uuid.UUID, workerID, providerStatus string, nextCheckAt time.Time) error {
	rows, err := r.queries.CompleteSenderIDSubmission(ctx, dbsqlc.CompleteSenderIDSubmissionParams{
		ID: id, WorkerID: strings.TrimSpace(workerID), ProviderStatus: strings.TrimSpace(providerStatus), NextCheckAt: pgconv.TimestamptzFromTime(nextCheckAt),
	})
	if err != nil {
		return fmt.Errorf("complete Sender ID registration claim: %w", err)
	}
	if rows != 1 {
		return ErrRegistrationClaimLost
	}
	return nil
}

func (r *Repository) CompleteStatus(ctx context.Context, id uuid.UUID, workerID, status, providerStatus string, whitelisted bool, rejectionReason *string, nextCheckAt time.Time) error {
	rows, err := r.queries.CompleteSenderIDStatus(ctx, dbsqlc.CompleteSenderIDStatusParams{
		ID: id, WorkerID: strings.TrimSpace(workerID), Status: strings.ToLower(strings.TrimSpace(status)), ProviderStatus: strings.TrimSpace(providerStatus), Whitelisted: whitelisted, RejectionReason: rejectionReason, NextCheckAt: pgconv.TimestamptzFromTime(nextCheckAt),
	})
	if err != nil {
		return fmt.Errorf("complete Sender ID provider status: %w", err)
	}
	if rows != 1 {
		return ErrRegistrationClaimLost
	}
	return nil
}

func (r *Repository) RecordProviderFailure(ctx context.Context, id uuid.UUID, workerID, providerStatus string, providerError error, nextCheckAt time.Time) error {
	message := "sender ID provider operation failed"
	if providerError != nil {
		message = providerError.Error()
	}
	rows, err := r.queries.RecordSenderIDProviderFailure(ctx, dbsqlc.RecordSenderIDProviderFailureParams{
		ID: id, WorkerID: strings.TrimSpace(workerID), ProviderStatus: strings.TrimSpace(providerStatus), LastError: &message, NextCheckAt: pgconv.TimestamptzFromTime(nextCheckAt),
	})
	if err != nil {
		return fmt.Errorf("record Sender ID provider failure: %w", err)
	}
	if rows != 1 {
		return ErrRegistrationClaimLost
	}
	return nil
}

func (r *Repository) ListNotificationRecipients(ctx context.Context, teamID uuid.UUID) ([]systemmail.Recipient, error) {
	rows, err := r.queries.ListActiveTeamOwnerRecipients(ctx, dbsqlc.ListActiveTeamOwnerRecipientsParams{TeamID: teamID})
	if err != nil {
		return nil, fmt.Errorf("list Sender ID notification recipients: %w", err)
	}
	recipients := make([]systemmail.Recipient, 0, len(rows))
	for _, row := range rows {
		recipients = append(recipients, systemmail.Recipient{Name: row.Name, Email: row.Email})
	}
	return recipients, nil
}

func senderIDFromRow(row dbsqlc.SenderID) SenderID {
	var createdBy *string
	if row.CreatedBy != nil {
		value := row.CreatedBy.String()
		createdBy = &value
	}
	purpose := ""
	if row.Purpose != nil {
		purpose = *row.Purpose
	}
	return SenderID{
		ID:              row.ID.String(),
		TeamID:          row.TeamID.String(),
		Name:            row.Name,
		CountryCode:     row.CountryCode,
		Purpose:         purpose,
		Status:          row.Status,
		Provider:        row.Provider,
		RejectionReason: row.RejectionReason,
		ApprovedAt:      pgconv.TimestamptzToTimePtr(row.ApprovedAt),
		RejectedAt:      pgconv.TimestamptzToTimePtr(row.RejectedAt),
		SuspendedAt:     pgconv.TimestamptzToTimePtr(row.SuspendedAt),
		CreatedBy:       createdBy,
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
