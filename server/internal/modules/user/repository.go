package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	dbsqlc "github.com/dugble/dugble/server/internal/database/sqlc"
	notifications "github.com/dugble/dugble/server/internal/platform/systemmail"
)

var errEmailInUse = errors.New("email is already in use")

const lockEmailChangeIdentifierSQL = `
SELECT pg_advisory_xact_lock(
    hashtextextended('verification-token:' || $1, 0)
)
`

type Repository struct {
	db      *pgxpool.Pool
	queries *dbsqlc.Queries
}

func (r *Repository) GetNotificationRecipient(ctx context.Context, userID uuid.UUID) (notifications.Recipient, error) {
	user, err := r.GetByID(ctx, userID.String())
	if err != nil {
		return notifications.Recipient{}, err
	}
	return notifications.Recipient{Name: user.Name, Email: user.Email}, nil
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db, queries: dbsqlc.New(db)}
}

func (r *Repository) GetByID(ctx context.Context, id string) (User, error) {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return User{}, fmt.Errorf("parse user id: %w", err)
	}

	row, err := r.queries.GetUserByID(ctx, dbsqlc.GetUserByIDParams{ID: parsedID})
	if err != nil {
		return User{}, fmt.Errorf("get user by id: %w", err)
	}

	return userFromSQLC(row), nil
}

func (r *Repository) GetPasswordHash(ctx context.Context, id string) (*string, error) {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}
	var passwordHash *string
	if err := r.db.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, parsedID).Scan(&passwordHash); err != nil {
		return nil, fmt.Errorf("get user password hash: %w", err)
	}
	return passwordHash, nil
}

func (r *Repository) UpdateProfile(ctx context.Context, id string, name string) (User, error) {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return User{}, fmt.Errorf("parse user id: %w", err)
	}

	row, err := r.queries.UpdateUserProfile(
		ctx,
		dbsqlc.UpdateUserProfileParams{ID: parsedID, Name: name},
	)
	if err != nil {
		return User{}, fmt.Errorf("update user profile: %w", err)
	}

	return userFromSQLC(row), nil
}

func (r *Repository) StartEmailChange(
	ctx context.Context,
	id string,
	pendingEmail string,
	identifier string,
	tokenHash string,
	expiresAt time.Time,
) (emailChangeRequest, error) {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return emailChangeRequest{}, fmt.Errorf("parse user id: %w", err)
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return emailChangeRequest{}, fmt.Errorf("begin email change: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, lockEmailChangeIdentifierSQL, identifier); err != nil {
		return emailChangeRequest{}, fmt.Errorf("lock email change: %w", err)
	}
	inUse, err := emailInUse(ctx, tx, pendingEmail, parsedID)
	if err != nil {
		return emailChangeRequest{}, err
	}
	if inUse {
		return emailChangeRequest{}, errEmailInUse
	}

	var request emailChangeRequest
	err = tx.QueryRow(ctx, `
INSERT INTO email_change_requests (user_id, pending_email, requested_at, expires_at)
VALUES ($1, $2, now(), $3)
ON CONFLICT (user_id) DO UPDATE
SET pending_email = EXCLUDED.pending_email,
    requested_at = now(),
    expires_at = EXCLUDED.expires_at
RETURNING user_id::text, pending_email, requested_at, expires_at
`, parsedID, pendingEmail, expiresAt).Scan(&request.UserID, &request.PendingEmail, &request.RequestedAt, &request.ExpiresAt)
	if err != nil {
		return emailChangeRequest{}, fmt.Errorf("store email change request: %w", err)
	}
	if err := replaceVerificationToken(ctx, tx, identifier, tokenHash, expiresAt); err != nil {
		return emailChangeRequest{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return emailChangeRequest{}, fmt.Errorf("commit email change request: %w", err)
	}
	return request, nil
}

func (r *Repository) ResendEmailChange(
	ctx context.Context,
	id string,
	identifier string,
	tokenHash string,
	expiresAt time.Time,
) (emailChangeRequest, error) {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return emailChangeRequest{}, fmt.Errorf("parse user id: %w", err)
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return emailChangeRequest{}, fmt.Errorf("begin email change resend: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, lockEmailChangeIdentifierSQL, identifier); err != nil {
		return emailChangeRequest{}, fmt.Errorf("lock email change: %w", err)
	}
	var request emailChangeRequest
	err = tx.QueryRow(ctx, `
UPDATE email_change_requests
SET expires_at = $2
WHERE user_id = $1
  AND expires_at > now()
RETURNING user_id::text, pending_email, requested_at, expires_at
`, parsedID, expiresAt).Scan(&request.UserID, &request.PendingEmail, &request.RequestedAt, &request.ExpiresAt)
	if err != nil {
		return emailChangeRequest{}, fmt.Errorf("load pending email change: %w", err)
	}
	inUse, err := emailInUse(ctx, tx, request.PendingEmail, parsedID)
	if err != nil {
		return emailChangeRequest{}, err
	}
	if inUse {
		return emailChangeRequest{}, errEmailInUse
	}
	if err := replaceVerificationToken(ctx, tx, identifier, tokenHash, expiresAt); err != nil {
		return emailChangeRequest{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return emailChangeRequest{}, fmt.Errorf("commit email change resend: %w", err)
	}
	return request, nil
}

func (r *Repository) CancelEmailChange(ctx context.Context, id string, identifier string) error {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("parse user id: %w", err)
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin email change cancellation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, lockEmailChangeIdentifierSQL, identifier); err != nil {
		return fmt.Errorf("lock email change: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM email_change_requests WHERE user_id = $1`, parsedID); err != nil {
		return fmt.Errorf("delete email change request: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM verification_tokens WHERE identifier = $1`, identifier); err != nil {
		return fmt.Errorf("delete email change tokens: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit email change cancellation: %w", err)
	}
	return nil
}

func (r *Repository) VerifyEmailChange(
	ctx context.Context,
	id string,
	identifier string,
	tokenHash string,
) (User, string, error) {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return User{}, "", fmt.Errorf("parse user id: %w", err)
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return User{}, "", fmt.Errorf("begin email change verification: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, lockEmailChangeIdentifierSQL, identifier); err != nil {
		return User{}, "", fmt.Errorf("lock email change: %w", err)
	}
	var pendingEmail string
	if err := tx.QueryRow(ctx, `
SELECT pending_email
FROM email_change_requests
WHERE user_id = $1
  AND expires_at > now()
FOR UPDATE
`, parsedID).Scan(&pendingEmail); err != nil {
		return User{}, "", fmt.Errorf("load pending email change: %w", err)
	}
	var consumed string
	if err := tx.QueryRow(ctx, `
DELETE FROM verification_tokens
WHERE identifier = $1
  AND token_hash = $2
  AND expires_at > now()
RETURNING identifier
`, identifier, tokenHash).Scan(&consumed); err != nil {
		return User{}, "", fmt.Errorf("consume email change token: %w", err)
	}
	inUse, err := emailInUse(ctx, tx, pendingEmail, parsedID)
	if err != nil {
		return User{}, "", err
	}
	if inUse {
		return User{}, "", errEmailInUse
	}

	var previousEmail string
	if err := tx.QueryRow(ctx, `SELECT email FROM users WHERE id = $1 FOR UPDATE`, parsedID).Scan(&previousEmail); err != nil {
		return User{}, "", fmt.Errorf("load current email: %w", err)
	}
	if _, err := tx.Exec(ctx, `
UPDATE users
SET email = $2,
    email_verified = true,
    credential_version = credential_version + 1,
    security_updated_at = now(),
    updated_at = now()
WHERE id = $1
`, parsedID, pendingEmail); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, "", errEmailInUse
		}
		return User{}, "", fmt.Errorf("promote pending email: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM email_change_requests WHERE user_id = $1`, parsedID); err != nil {
		return User{}, "", fmt.Errorf("delete email change request: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM verification_tokens WHERE identifier = $1`, identifier); err != nil {
		return User{}, "", fmt.Errorf("delete email change tokens: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`, parsedID); err != nil {
		return User{}, "", fmt.Errorf("revoke user sessions: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, "", fmt.Errorf("commit email change verification: %w", err)
	}
	updated, err := r.GetByID(ctx, id)
	if err != nil {
		return User{}, "", err
	}
	return updated, previousEmail, nil
}

func replaceVerificationToken(ctx context.Context, tx pgx.Tx, identifier, tokenHash string, expiresAt time.Time) error {
	if _, err := tx.Exec(ctx, `DELETE FROM verification_tokens WHERE identifier = $1`, identifier); err != nil {
		return fmt.Errorf("delete superseded email change tokens: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO verification_tokens (identifier, token_hash, expires_at)
VALUES ($1, $2, $3)
`, identifier, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("create email change token: %w", err)
	}
	return nil
}

func emailInUse(ctx context.Context, tx pgx.Tx, email string, excludeUserID uuid.UUID) (bool, error) {
	var inUse bool
	if err := tx.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1
    FROM users
    WHERE lower(email) = lower($1)
      AND id <> $2
)
`, email, excludeUserID).Scan(&inUse); err != nil {
		return false, fmt.Errorf("check email availability: %w", err)
	}
	return inUse, nil
}

func (r *Repository) UpdatePassword(
	ctx context.Context,
	id string,
	passwordHash string,
) (User, error) {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return User{}, fmt.Errorf("parse user id: %w", err)
	}

	row, err := r.queries.UpdateUserPassword(
		ctx,
		dbsqlc.UpdateUserPasswordParams{ID: parsedID, PasswordHash: &passwordHash},
	)
	if err != nil {
		return User{}, fmt.Errorf("update user password: %w", err)
	}

	return userFromSQLC(row), nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("parse user id: %w", err)
	}

	return r.queries.DeleteUser(ctx, dbsqlc.DeleteUserParams{ID: parsedID})
}

func userFromSQLC(row dbsqlc.User) User {
	return User{
		ID:            row.ID.String(),
		Email:         row.Email,
		EmailVerified: row.EmailVerified,
		Name:          row.Name,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}
}
