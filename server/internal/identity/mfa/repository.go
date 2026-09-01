package mfa

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	dbsqlc "github.com/dugble/dugble/server/internal/database/sqlc"
	"github.com/dugble/dugble/server/pkg/pgconv"
)

type Repository struct {
	queries *dbsqlc.Queries
}

func NewRepository(queries *dbsqlc.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) WithTx(tx pgx.Tx) *Repository {
	return NewRepository(r.queries.WithTx(tx))
}

func (r *Repository) PutUnverified(ctx context.Context, userID uuid.UUID, ciphertext []byte) error {
	rows, err := r.queries.PutUnverifiedTOTPCredential(ctx, dbsqlc.PutUnverifiedTOTPCredentialParams{
		UserID: userID, SecretCiphertext: ciphertext,
	})
	if err != nil {
		return err
	}
	if rows != 1 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) GetCredential(ctx context.Context, userID uuid.UUID) (Credential, error) {
	row, err := r.queries.GetTOTPCredential(ctx, dbsqlc.GetTOTPCredentialParams{UserID: userID})
	if err != nil {
		return Credential{}, err
	}
	var verifiedAt *time.Time
	if row.VerifiedAt.Valid {
		verifiedAt = &row.VerifiedAt.Time
	}
	return Credential{SecretCiphertext: row.SecretCiphertext, VerifiedAt: verifiedAt, LastUsedStep: row.LastUsedStep}, nil
}

func (r *Repository) RotateSecretCiphertext(ctx context.Context, userID uuid.UUID, oldCiphertext, newCiphertext []byte) error {
	return r.queries.RotateTOTPSecretCiphertext(ctx, dbsqlc.RotateTOTPSecretCiphertextParams{UserID: userID, OldCiphertext: oldCiphertext, NewCiphertext: newCiphertext})
}

func (r *Repository) Confirm(ctx context.Context, userID uuid.UUID, sessionID string, step int64, codeHashes []string) error {
	queries := r.queries
	var err error

	rows, err := queries.ConfirmTOTPCredential(ctx, dbsqlc.ConfirmTOTPCredentialParams{UserID: userID, LastUsedStep: &step})
	if err != nil {
		return err
	}
	if rows != 1 {
		return pgx.ErrNoRows
	}
	if err = queries.DeleteRecoveryCodes(ctx, dbsqlc.DeleteRecoveryCodesParams{UserID: userID}); err != nil {
		return err
	}
	for _, hash := range codeHashes {
		if err = queries.CreateRecoveryCode(ctx, dbsqlc.CreateRecoveryCodeParams{UserID: userID, CodeHash: hash}); err != nil {
			return err
		}
	}
	rows, err = queries.ElevateSessionAfterMFAEnrollment(ctx, dbsqlc.ElevateSessionAfterMFAEnrollmentParams{SessionID: sessionID, UserID: userID})
	if err != nil {
		return err
	}
	if rows != 1 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) Verify(ctx context.Context, userID uuid.UUID, sessionID string, step int64) error {
	rows, err := r.queries.VerifyTOTPAndElevateSession(ctx, dbsqlc.VerifyTOTPAndElevateSessionParams{
		UserID: userID, SessionID: sessionID, LastUsedStep: &step,
	})
	if err != nil {
		return err
	}
	if rows != 1 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) UseRecoveryCode(ctx context.Context, userID uuid.UUID, sessionID, codeHash string) error {
	rows, err := r.queries.UseRecoveryCodeAndElevateSession(ctx, dbsqlc.UseRecoveryCodeAndElevateSessionParams{
		UserID: userID, SessionID: sessionID, CodeHash: codeHash,
	})
	if err != nil {
		return err
	}
	if rows != 1 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) Disable(ctx context.Context, userID uuid.UUID, currentSessionID string) error {
	queries := r.queries
	var err error
	if err = queries.DeleteTOTPCredential(ctx, dbsqlc.DeleteTOTPCredentialParams{UserID: userID}); err != nil {
		return err
	}
	if err = queries.DeleteRecoveryCodes(ctx, dbsqlc.DeleteRecoveryCodesParams{UserID: userID}); err != nil {
		return err
	}
	params := dbsqlc.RevokeOtherSessionsAfterMFADisableParams{UserID: userID, CurrentSessionID: currentSessionID}
	if err = queries.RevokeOtherSessionsAfterMFADisable(ctx, params); err != nil {
		return err
	}
	if err = queries.DowngradeSessionAfterMFADisable(ctx, dbsqlc.DowngradeSessionAfterMFADisableParams(params)); err != nil {
		return err
	}
	return nil
}

func (r *Repository) Enabled(ctx context.Context, userID uuid.UUID) (bool, error) {
	return r.queries.IsTOTPEnabled(ctx, dbsqlc.IsTOTPEnabledParams{UserID: userID})
}

func (r *Repository) CreateLoginChallenge(ctx context.Context, tokenHash string, userID uuid.UUID, credentialVersion int64, expiresAt time.Time) error {
	return r.queries.CreateMFALoginChallenge(ctx, dbsqlc.CreateMFALoginChallengeParams{
		TokenHash:         tokenHash,
		UserID:            &userID,
		CredentialVersion: credentialVersion,
		ExpiresAt:         pgconv.TimestamptzFromTime(expiresAt),
	})
}

func (r *Repository) GetLoginChallenge(ctx context.Context, tokenHash string) (uuid.UUID, Credential, error) {
	row, err := r.queries.GetActiveMFALoginChallenge(ctx, dbsqlc.GetActiveMFALoginChallengeParams{TokenHash: tokenHash})
	if err != nil {
		return uuid.Nil, Credential{}, err
	}
	return *row.UserID, Credential{SecretCiphertext: row.SecretCiphertext, LastUsedStep: row.LastUsedStep}, nil
}

func (r *Repository) ConsumeLoginTOTP(ctx context.Context, tokenHash string, userID uuid.UUID, step int64) error {
	rows, err := r.queries.ConsumeMFALoginChallengeWithTOTP(ctx, dbsqlc.ConsumeMFALoginChallengeWithTOTPParams{TokenHash: tokenHash, UserID: &userID, LastUsedStep: &step})
	if err != nil {
		return err
	}
	if rows != 1 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) ConsumeLoginRecoveryCode(ctx context.Context, tokenHash string, userID uuid.UUID, codeHash string) error {
	rows, err := r.queries.ConsumeMFALoginChallengeWithRecoveryCode(ctx, dbsqlc.ConsumeMFALoginChallengeWithRecoveryCodeParams{TokenHash: tokenHash, UserID: &userID, CodeHash: codeHash})
	if err != nil {
		return err
	}
	if rows != 1 {
		return pgx.ErrNoRows
	}
	return nil
}
