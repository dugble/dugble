package mfa

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	sentrymonitoring "github.com/dugble/dugble/server/internal/integrations/monitoring/sentry"
	notifications "github.com/dugble/dugble/server/internal/messaging/email/systemmail"
	"github.com/dugble/dugble/server/internal/modules/audit"
	"github.com/dugble/dugble/server/internal/security"
	"github.com/dugble/dugble/server/internal/security/authn"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const recoveryCodeCount = 10
const loginChallengeTTL = 5 * time.Minute
const loginChallengePrefix = "dgb_mfa_"

type store interface {
	PutUnverified(context.Context, uuid.UUID, []byte) error
	GetCredential(context.Context, uuid.UUID) (Credential, error)
	RotateSecretCiphertext(context.Context, uuid.UUID, []byte, []byte) error
	Confirm(context.Context, uuid.UUID, string, int64, []string) error
	Verify(context.Context, uuid.UUID, string, int64) error
	UseRecoveryCode(context.Context, uuid.UUID, string, string) error
	Disable(context.Context, uuid.UUID, string) error
	Enabled(context.Context, uuid.UUID) (bool, error)
	CreateLoginChallenge(context.Context, string, uuid.UUID, int64, time.Time) error
	GetLoginChallenge(context.Context, string) (uuid.UUID, Credential, error)
	ConsumeLoginTOTP(context.Context, string, uuid.UUID, int64) error
	ConsumeLoginRecoveryCode(context.Context, string, uuid.UUID, string) error
}

type Service struct {
	db         *pgxpool.Pool
	repository store
	cipher     *security.SecretCipher
	issuer     string
	now        func() time.Time
	notifier   SecurityNotifier
	recipients RecipientStore
}

type SecurityNotifier interface {
	SendMFAEnabled(context.Context, notifications.SendSecurityEventInput) error
	SendMFADisabled(context.Context, notifications.SendSecurityEventInput) error
	SendRecoveryCodeUsed(context.Context, notifications.SendSecurityEventInput) error
	SendMFALoginFailed(context.Context, notifications.SendSecurityEventInput) error
}

type RecipientStore interface {
	GetNotificationRecipient(context.Context, uuid.UUID) (notifications.Recipient, error)
}

func NewService(db *pgxpool.Pool, repository store, cipher *security.SecretCipher, issuer string) *Service {
	return &Service{repository: repository, cipher: cipher, issuer: issuer, now: time.Now}
}

func (s *Service) WithNotifier(notifier SecurityNotifier) *Service {
	s.notifier = notifier
	return s
}

func (s *Service) WithRecipientStore(recipients RecipientStore) *Service {
	s.recipients = recipients
	return s
}

func (s *Service) Enroll(ctx context.Context) (EnrollResponse, error) {
	principal, err := principalFromContext(ctx)
	if err != nil {
		return EnrollResponse{}, err
	}
	secret, err := NewTOTPSecret()
	if err != nil {
		return EnrollResponse{}, apperrors.NewInternal("Unable to generate MFA secret", err)
	}
	ciphertext, err := s.cipher.Encrypt([]byte(secret))
	if err != nil {
		return EnrollResponse{}, apperrors.NewInternal("Unable to protect MFA secret", err)
	}
	if err := s.repository.PutUnverified(ctx, principal.UserID, ciphertext); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return EnrollResponse{}, apperrors.NewConflict("MFA is already enabled")
		}
		return EnrollResponse{}, apperrors.NewInternal("Unable to begin MFA enrollment", err)
	}
	return EnrollResponse{Secret: secret, URI: TOTPURI(s.issuer, principal.Email, secret)}, nil
}

func (s *Service) transactionRepository(ctx context.Context) (*Repository, pgx.Tx, error) {
	repository, ok := s.repository.(*Repository)
	if !ok {
		return nil, nil, errors.New("MFA repository does not support transactions")
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	return repository.WithTx(tx), tx, nil
}
func (s *Service) confirm(ctx context.Context, userID uuid.UUID, sessionID string, step int64, hashes []string) error {
	repository, tx, err := s.transactionRepository(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = repository.Confirm(ctx, userID, sessionID, step, hashes); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (s *Service) disable(ctx context.Context, userID uuid.UUID, sessionID string) error {
	repository, tx, err := s.transactionRepository(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = repository.Disable(ctx, userID, sessionID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) Confirm(ctx context.Context, code string) (ConfirmResponse, error) {
	principal, err := principalFromContext(ctx)
	if err != nil {
		return ConfirmResponse{}, err
	}
	credential, err := s.repository.GetCredential(ctx, principal.UserID)
	if err != nil || credential.VerifiedAt != nil {
		return ConfirmResponse{}, apperrors.NewBadRequest("MFA enrollment is not pending")
	}
	step, ok := s.validateCredential(ctx, principal.UserID, credential, code)
	if !ok {
		return ConfirmResponse{}, apperrors.NewUnauthorized("Invalid authentication code")
	}
	codes, hashes, err := newRecoveryCodes()
	if err != nil {
		return ConfirmResponse{}, apperrors.NewInternal("Unable to create recovery codes", err)
	}
	if err := s.confirm(ctx, principal.UserID, principal.SessionID, step, hashes); err != nil {
		return ConfirmResponse{}, apperrors.NewInternal("Unable to confirm MFA enrollment", err)
	}
	audit.RecordIdentity(ctx, principal.UserID, audit.Event{Action: "identity.mfa_enabled", ResourceType: "user", ResourceID: principal.UserID.String()})
	s.notify(ctx, principal, "enabled")
	return ConfirmResponse{RecoveryCodes: codes}, nil
}

func (s *Service) Verify(ctx context.Context, code string) error {
	principal, err := principalFromContext(ctx)
	if err != nil {
		return err
	}
	credential, err := s.repository.GetCredential(ctx, principal.UserID)
	if err != nil || credential.VerifiedAt == nil {
		return apperrors.NewBadRequest("MFA is not enabled")
	}
	step, ok := s.validateCredential(ctx, principal.UserID, credential, code)
	if !ok {
		return apperrors.NewUnauthorized("Invalid authentication code")
	}
	if credential.LastUsedStep != nil && step <= *credential.LastUsedStep {
		return apperrors.NewUnauthorized("Authentication code was already used")
	}
	if err := s.repository.Verify(ctx, principal.UserID, principal.SessionID, step); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperrors.NewUnauthorized("Authentication code was already used")
		}
		return apperrors.NewInternal("Unable to verify authentication code", err)
	}
	audit.RecordIdentity(ctx, principal.UserID, audit.Event{Action: "identity.mfa_verified", ResourceType: "session", ResourceID: principal.SessionID})
	return nil
}

func (s *Service) Recover(ctx context.Context, code string) error {
	principal, err := principalFromContext(ctx)
	if err != nil {
		return err
	}
	code, err = validateRecoveryCode(code)
	if err != nil {
		return err
	}
	hash := HashRecoveryCode(code)
	if err := s.repository.UseRecoveryCode(ctx, principal.UserID, principal.SessionID, hash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperrors.NewUnauthorized("Invalid or used recovery code")
		}
		return apperrors.NewInternal("Unable to use recovery code", err)
	}
	audit.RecordIdentity(ctx, principal.UserID, audit.Event{Action: "identity.recovery_code_used", ResourceType: "session", ResourceID: principal.SessionID})
	s.notify(ctx, principal, "recovery")
	return nil
}

func (s *Service) Disable(ctx context.Context) error {
	principal, err := principalFromContext(ctx)
	if err != nil {
		return err
	}
	if err := s.disable(ctx, principal.UserID, principal.SessionID); err != nil {
		return apperrors.NewInternal("Unable to disable MFA", err)
	}
	audit.RecordIdentity(ctx, principal.UserID, audit.Event{Action: "identity.mfa_disabled", ResourceType: "user", ResourceID: principal.UserID.String()})
	s.notify(ctx, principal, "disabled")
	return nil
}

func (s *Service) notify(ctx context.Context, principal authn.Principal, event string) {
	if s.notifier == nil || strings.TrimSpace(principal.Email) == "" {
		return
	}
	input := notifications.SendSecurityEventInput{ToEmail: principal.Email, Name: principal.Name}
	var err error
	switch event {
	case "enabled":
		err = s.notifier.SendMFAEnabled(ctx, input)
	case "disabled":
		err = s.notifier.SendMFADisabled(ctx, input)
	case "recovery":
		err = s.notifier.SendRecoveryCodeUsed(ctx, input)
	}
	if err != nil {
		sentrymonitoring.Warn("failed to send MFA security notification", "error", err, "event", event, "user_id", principal.UserID)
	}
}

func (s *Service) Status(ctx context.Context) (StatusResponse, error) {
	principal, err := principalFromContext(ctx)
	if err != nil {
		return StatusResponse{}, err
	}
	enabled, err := s.repository.Enabled(ctx, principal.UserID)
	if err != nil {
		return StatusResponse{}, apperrors.NewInternal("Unable to get MFA status", err)
	}
	return StatusResponse{Enabled: enabled}, nil
}

func (s *Service) validateCredential(ctx context.Context, userID uuid.UUID, credential Credential, code string) (int64, bool) {
	secret, replacement, rotated, err := s.cipher.DecryptAndRotate(credential.SecretCiphertext)
	if err != nil {
		return 0, false
	}
	if rotated {
		_ = s.repository.RotateSecretCiphertext(ctx, userID, credential.SecretCiphertext, replacement)
	}
	return ValidateTOTP(string(secret), normalizeAuthenticationCode(code), s.now().UTC())
}

func principalFromContext(ctx context.Context) (authn.Principal, error) {
	principal, ok := authn.PrincipalFromContext(ctx)
	if !ok {
		return authn.Principal{}, apperrors.NewUnauthorized("Authentication is required")
	}
	return principal, nil
}

func newRecoveryCodes() ([]string, []string, error) {
	codes := make([]string, 0, recoveryCodeCount)
	hashes := make([]string, 0, recoveryCodeCount)
	for range recoveryCodeCount {
		code, err := NewRecoveryCode()
		if err != nil {
			return nil, nil, err
		}
		codes = append(codes, code)
		hashes = append(hashes, HashRecoveryCode(code))
	}
	return codes, hashes, nil
}

func (s *Service) LoginEnabled(ctx context.Context, userID uuid.UUID) (bool, error) {
	return s.repository.Enabled(ctx, userID)
}

func (s *Service) BeginLogin(ctx context.Context, userID uuid.UUID, credentialVersion int64) (string, error) {
	value, err := security.NewSessionToken()
	if err != nil {
		return "", err
	}
	token := loginChallengePrefix + value
	if err := s.repository.CreateLoginChallenge(ctx, security.HashSessionToken(token), userID, credentialVersion, s.now().UTC().Add(loginChallengeTTL)); err != nil {
		return "", err
	}
	return token, nil
}

func (s *Service) CompleteLoginTOTP(ctx context.Context, challengeToken, code string) (uuid.UUID, error) {
	tokenHash, err := validateLoginChallengeToken(challengeToken)
	if err != nil {
		return uuid.Nil, err
	}
	userID, credential, err := s.repository.GetLoginChallenge(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, pgx.ErrNoRows
		}
		return uuid.Nil, err
	}
	step, ok := s.validateCredential(ctx, userID, credential, code)
	if !ok || (credential.LastUsedStep != nil && step <= *credential.LastUsedStep) {
		s.notifyFailedLogin(ctx, userID)
		return uuid.Nil, pgx.ErrNoRows
	}
	if err := s.repository.ConsumeLoginTOTP(ctx, tokenHash, userID, step); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.notifyFailedLogin(ctx, userID)
		}
		return uuid.Nil, err
	}
	return userID, nil
}

func (s *Service) CompleteLoginRecovery(ctx context.Context, challengeToken, code string) (uuid.UUID, error) {
	tokenHash, err := validateLoginChallengeToken(challengeToken)
	if err != nil {
		return uuid.Nil, pgx.ErrNoRows
	}
	code, err = validateRecoveryCode(code)
	if err != nil {
		return uuid.Nil, pgx.ErrNoRows
	}
	userID, _, err := s.repository.GetLoginChallenge(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, pgx.ErrNoRows
		}
		return uuid.Nil, err
	}
	if err := s.repository.ConsumeLoginRecoveryCode(ctx, tokenHash, userID, HashRecoveryCode(code)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.notifyFailedLogin(ctx, userID)
		}
		return uuid.Nil, err
	}
	return userID, nil
}

func (s *Service) notifyFailedLogin(ctx context.Context, userID uuid.UUID) {
	if s.notifier == nil || s.recipients == nil {
		return
	}
	recipient, err := s.recipients.GetNotificationRecipient(ctx, userID)
	if err != nil {
		sentrymonitoring.Warn("failed to resolve failed MFA notification recipient", "error", err, "user_id", userID)
		return
	}
	if err := s.notifier.SendMFALoginFailed(ctx, notifications.SendSecurityEventInput{ToEmail: recipient.Email, Name: recipient.Name}); err != nil {
		sentrymonitoring.Warn("failed to send failed MFA notification", "error", err, "user_id", userID)
	}
}

func NewTOTPSecret() (string, error) {
	value := make([]byte, 20)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(value), nil
}

func TOTPURI(issuer, account, secret string) string {
	values := url.Values{
		"secret": {secret},
		"issuer": {issuer},
		"period": {"30"},
		"digits": {"6"},
	}
	return "otpauth://totp/" + url.PathEscape(issuer+":"+account) + "?" + values.Encode()
}

func ValidateTOTP(secret, code string, now time.Time) (int64, bool) {
	code = strings.TrimSpace(code)
	if len(code) != 6 || strings.IndexFunc(code, func(value rune) bool {
		return value < '0' || value > '9'
	}) >= 0 {
		return 0, false
	}
	step := now.Unix() / 30
	for offset := int64(-1); offset <= 1; offset++ {
		expected := totpCode(secret, step+offset)
		if expected != "" && hmac.Equal([]byte(expected), []byte(code)) {
			return step + offset, true
		}
	}
	return 0, false
}

func totpCode(secret string, step int64) string {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return ""
	}
	var counter [8]byte
	binary.BigEndian.PutUint64(counter[:], uint64(step))
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(counter[:])
	digest := mac.Sum(nil)
	offset := digest[len(digest)-1] & 15
	value := (uint32(digest[offset])&127)<<24 |
		uint32(digest[offset+1])<<16 |
		uint32(digest[offset+2])<<8 |
		uint32(digest[offset+3])
	return fmt.Sprintf("%06d", value%1_000_000)
}

func NewRecoveryCode() (string, error) {
	value := make([]byte, 10)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(value)
	return encoded[:8] + "-" + encoded[8:], nil
}

func HashRecoveryCode(code string) string {
	value := strings.ReplaceAll(strings.ToUpper(strings.TrimSpace(code)), "-", "")
	digest := sha256.Sum256([]byte(value))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}
