package user

import (
	"context"
	"errors"
	"strings"
	"time"

	sentrymonitoring "github.com/dugble/dugble/server/internal/adapters/monitoring/sentry"

	"github.com/jackc/pgx/v5"

	"github.com/dugble/dugble/server/internal/authn"
	notifications "github.com/dugble/dugble/server/internal/platform/systemmail"
	"github.com/dugble/dugble/server/internal/security"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

const emailChangeTokenTTL = 24 * time.Hour

type Service struct {
	repository *Repository
	notifier   SecurityNotifier
}

type SecurityNotifier interface {
	SendPasswordChanged(context.Context, notifications.SendPasswordChangedInput) error
	SendEmailChanged(context.Context, notifications.SendEmailChangedInput) error
	SendEmailChangeVerification(context.Context, notifications.SendEmailChangeVerificationInput) error
	SendAccountDeleted(context.Context, notifications.SendSecurityEventInput) error
}

func NewService(repository *Repository, notifiers ...SecurityNotifier) *Service {
	service := &Service{repository: repository}
	if len(notifiers) > 0 {
		service.notifier = notifiers[0]
	}
	return service
}

func (s *Service) GetMe(ctx context.Context) (User, error) {
	principal, ok := authn.PrincipalFromContext(ctx)
	if !ok {
		return User{}, apperrors.NewUnauthorized("Authentication is required")
	}
	return s.GetByID(ctx, principal.UserID.String())
}

func (s *Service) GetByID(ctx context.Context, id string) (User, error) {
	id, err := validateID(id)
	if err != nil {
		return User{}, err
	}

	user, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return User{}, apperrors.NewNotFound("User not found")
	}

	return user, nil
}

func (s *Service) UpdateProfile(ctx context.Context, req UpdateProfileRequest) (User, error) {
	principal, ok := authn.PrincipalFromContext(ctx)
	if !ok {
		return User{}, apperrors.NewUnauthorized("Authentication is required")
	}

	name, err := validateName(req.Name)
	if err != nil {
		return User{}, err
	}

	updated, err := s.repository.UpdateProfile(ctx, principal.UserID.String(), name)
	if err != nil {
		return User{}, apperrors.NewInternal("Unable to update profile", err)
	}

	return updated, nil
}

func (s *Service) UpdateEmail(ctx context.Context, req UpdateEmailRequest) (EmailChangePending, error) {
	principal, ok := authn.PrincipalFromContext(ctx)
	if !ok {
		return EmailChangePending{}, apperrors.NewUnauthorized("Authentication is required")
	}

	email, err := validateEmail(req.Email)
	if err != nil {
		return EmailChangePending{}, err
	}
	currentPassword, err := validateCurrentPassword(req.CurrentPassword)
	if err != nil {
		return EmailChangePending{}, err
	}

	current, err := s.repository.GetByID(ctx, principal.UserID.String())
	if err != nil {
		return EmailChangePending{}, apperrors.NewInternal("Unable to load current email", err)
	}
	if strings.EqualFold(current.Email, email) {
		return EmailChangePending{}, apperrors.NewBadRequest("New email must be different from current email")
	}
	passwordHash, err := s.repository.GetPasswordHash(ctx, principal.UserID.String())
	if err != nil {
		return EmailChangePending{}, apperrors.NewInternal("Unable to confirm current password", err)
	}
	if !security.VerifyPassword(passwordHash, currentPassword) {
		return EmailChangePending{}, apperrors.NewUnauthorized("Current password is incorrect")
	}

	token, expiresAt, err := newEmailChangeToken()
	if err != nil {
		return EmailChangePending{}, err
	}
	request, err := s.repository.StartEmailChange(
		ctx,
		principal.UserID.String(),
		email,
		emailChangeIdentifier(principal.UserID.String()),
		security.HashSessionToken(token),
		expiresAt,
	)
	if err != nil {
		if errors.Is(err, errEmailInUse) {
			return EmailChangePending{}, apperrors.NewConflict("Email is already in use")
		}
		return EmailChangePending{}, apperrors.NewInternal("Unable to start email change", err)
	}
	if s.notifier != nil {
		if err := s.notifier.SendEmailChangeVerification(ctx, notifications.SendEmailChangeVerificationInput{
			ToEmail: request.PendingEmail,
			Name:    current.Name,
			Token:   token,
		}); err != nil {
			return EmailChangePending{}, apperrors.NewInternal("Unable to deliver email change verification", err)
		}
	}
	return EmailChangePending{
		Email:                 current.Email,
		PendingEmail:          request.PendingEmail,
		VerificationExpiresAt: request.ExpiresAt,
	}, nil
}

func (s *Service) VerifyEmailChange(ctx context.Context, req VerifyEmailChangeRequest) (User, error) {
	principal, ok := authn.PrincipalFromContext(ctx)
	if !ok {
		return User{}, apperrors.NewUnauthorized("Authentication is required")
	}
	token, err := validateVerificationToken(req.Token)
	if err != nil {
		return User{}, err
	}
	updated, previousEmail, err := s.repository.VerifyEmailChange(
		ctx,
		principal.UserID.String(),
		emailChangeIdentifier(principal.UserID.String()),
		security.HashSessionToken(token),
	)
	if err != nil {
		if errors.Is(err, errEmailInUse) {
			return User{}, apperrors.NewConflict("Email is already in use")
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, apperrors.NewBadRequest("Email change token is invalid or expired")
		}
		return User{}, apperrors.NewInternal("Unable to verify email change", err)
	}

	if s.notifier != nil {
		for _, recipient := range uniqueEmails(previousEmail, updated.Email) {
			if err := s.notifier.SendEmailChanged(ctx, notifications.SendEmailChangedInput{
				ToEmail: recipient,
				Name:    updated.Name,
				Email:   updated.Email,
			}); err != nil {
				sentrymonitoring.Warn("failed to send email changed notification", "error", err, "user_id", updated.ID)
			}
		}
	}
	return updated, nil
}

func (s *Service) ResendEmailChange(ctx context.Context) (EmailChangePending, error) {
	principal, ok := authn.PrincipalFromContext(ctx)
	if !ok {
		return EmailChangePending{}, apperrors.NewUnauthorized("Authentication is required")
	}
	current, err := s.repository.GetByID(ctx, principal.UserID.String())
	if err != nil {
		return EmailChangePending{}, apperrors.NewInternal("Unable to load current email", err)
	}
	token, expiresAt, err := newEmailChangeToken()
	if err != nil {
		return EmailChangePending{}, err
	}
	request, err := s.repository.ResendEmailChange(
		ctx,
		principal.UserID.String(),
		emailChangeIdentifier(principal.UserID.String()),
		security.HashSessionToken(token),
		expiresAt,
	)
	if err != nil {
		if errors.Is(err, errEmailInUse) {
			return EmailChangePending{}, apperrors.NewConflict("Email is already in use")
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return EmailChangePending{}, apperrors.NewBadRequest("No pending email change")
		}
		return EmailChangePending{}, apperrors.NewInternal("Unable to resend email change verification", err)
	}
	if s.notifier != nil {
		if err := s.notifier.SendEmailChangeVerification(ctx, notifications.SendEmailChangeVerificationInput{
			ToEmail: request.PendingEmail,
			Name:    current.Name,
			Token:   token,
		}); err != nil {
			return EmailChangePending{}, apperrors.NewInternal("Unable to deliver email change verification", err)
		}
	}
	return EmailChangePending{
		Email:                 current.Email,
		PendingEmail:          request.PendingEmail,
		VerificationExpiresAt: request.ExpiresAt,
	}, nil
}

func newEmailChangeToken() (string, time.Time, error) {
	token, err := security.NewSessionToken()
	if err != nil {
		return "", time.Time{}, apperrors.NewInternal("Unable to create email change token", err)
	}
	return token, time.Now().UTC().Add(emailChangeTokenTTL), nil
}

func (s *Service) UpdatePassword(ctx context.Context, req UpdatePasswordRequest) (User, error) {
	principal, ok := authn.PrincipalFromContext(ctx)
	if !ok {
		return User{}, apperrors.NewUnauthorized("Authentication is required")
	}

	password, err := validatePassword(req.Password)
	if err != nil {
		return User{}, err
	}

	hash, err := security.HashPassword(password)
	if err != nil {
		return User{}, apperrors.NewInternal("Unable to hash password", err)
	}

	updated, err := s.repository.UpdatePassword(ctx, principal.UserID.String(), hash)
	if err != nil {
		return User{}, apperrors.NewInternal("Unable to update password", err)
	}

	if s.notifier != nil {
		if err := s.notifier.SendPasswordChanged(ctx, notifications.SendPasswordChangedInput{ToEmail: updated.Email, Name: updated.Name}); err != nil {
			sentrymonitoring.Warn("failed to send password changed notification", "error", err, "user_id", updated.ID)
		}
	}
	return updated, nil
}

func (s *Service) DeleteMe(ctx context.Context) error {
	principal, ok := authn.PrincipalFromContext(ctx)
	if !ok {
		return apperrors.NewUnauthorized("Authentication is required")
	}

	current, err := s.repository.GetByID(ctx, principal.UserID.String())
	if err != nil {
		return apperrors.NewInternal("Unable to load user", err)
	}
	if err := s.repository.Delete(ctx, principal.UserID.String()); err != nil {
		return apperrors.NewInternal("Unable to delete user", err)
	}

	if s.notifier != nil {
		if err := s.notifier.SendAccountDeleted(ctx, notifications.SendSecurityEventInput{ToEmail: current.Email, Name: current.Name}); err != nil {
			sentrymonitoring.Warn("failed to send account deleted notification", "error", err, "user_id", current.ID)
		}
	}
	return nil
}
