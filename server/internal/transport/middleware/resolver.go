package middleware

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dugble/dugble/server/internal/authn"
	"github.com/dugble/dugble/server/internal/authz"
	"github.com/dugble/dugble/server/internal/modules/session"
	"github.com/dugble/dugble/server/internal/modules/teamtoken"
	"github.com/dugble/dugble/server/internal/security"
)

// CredentialResolver normalizes every supported credential into an authn.Principal.
// Bearer credentials take precedence over session credentials.
type CredentialResolver struct {
	Sessions SessionStore
	Users    PrincipalRepository
	Tokens   TeamTokenStore
	Now      func() time.Time
}

type resolutionError struct{ err error }

func (err resolutionError) Error() string { return err.err.Error() }
func (err resolutionError) Unwrap() error { return err.err }

func (resolver CredentialResolver) Resolve(ctx context.Context, credential authn.Credential) (authn.Principal, error) {
	credential = credential.Normalize()
	if credential.BearerToken != "" {
		return resolver.resolveTeamToken(ctx, credential.BearerToken)
	}
	if credential.SessionToken != "" {
		return resolver.resolveSession(ctx, credential.SessionToken)
	}
	return authn.Principal{}, authn.ErrUnauthenticated
}

func (resolver CredentialResolver) resolveSession(ctx context.Context, secret string) (authn.Principal, error) {
	if resolver.Sessions == nil || resolver.Users == nil {
		return authn.Principal{}, resolutionError{errors.New("session authentication is not configured")}
	}
	record, err := resolver.Sessions.GetByTokenHash(ctx, security.HashSessionToken(secret))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return authn.Principal{}, authn.ErrUnauthenticated
		}
		return authn.Principal{}, resolutionError{fmt.Errorf("load session: %w", err)}
	}
	now := resolver.now()
	if record.RevokedAt != nil || !record.ExpiresAt.After(now) {
		return authn.Principal{}, authn.ErrUnauthenticated
	}
	principal, err := resolver.Users.GetPrincipalByUserID(ctx, record.UserID.String())
	if err != nil {
		return authn.Principal{}, resolutionError{fmt.Errorf("resolve session user: %w", err)}
	}
	if record.CredentialVersion != principal.CredentialVersion {
		return authn.Principal{}, authn.ErrUnauthenticated
	}
	if err := resolver.Sessions.Touch(ctx, record.ID); err != nil {
		return authn.Principal{}, resolutionError{fmt.Errorf("update session activity: %w", err)}
	}
	principal.Kind = authn.PrincipalUserSession
	principal.SubjectID = principal.UserID
	principal.SessionID = record.ID
	principal.Scopes = authz.FirstPartyUserSessionScopes()
	principal.AuthenticationMethod = record.Method
	principal.AssuranceLevel = record.Assurance
	principal.AuthenticatedAt = record.AuthenticatedAt
	principal.MFACompletedAt = record.MFACompletedAt
	return principal, nil
}

func (resolver CredentialResolver) resolveTeamToken(ctx context.Context, secret string) (authn.Principal, error) {
	if resolver.Tokens == nil {
		return authn.Principal{}, resolutionError{errors.New("team token authentication is not configured")}
	}
	if !strings.HasPrefix(secret, teamtoken.TokenPrefix) {
		return authn.Principal{}, authn.ErrUnauthenticated
	}
	token, err := resolver.Tokens.GetActiveByTokenHash(ctx, security.HashSessionToken(secret))
	if err != nil || token.RevokedAt != nil || (token.ExpiresAt != nil && !token.ExpiresAt.After(resolver.now())) {
		return authn.Principal{}, authn.ErrUnauthenticated
	}
	permissions, ok := tokenPermissions(token.Permissions)
	if !ok {
		return authn.Principal{}, authn.ErrUnauthenticated
	}
	tokenID, err := uuid.Parse(token.ID)
	if err != nil {
		return authn.Principal{}, authn.ErrUnauthenticated
	}
	teamID, err := uuid.Parse(token.TeamID)
	if err != nil {
		return authn.Principal{}, authn.ErrUnauthenticated
	}
	scopes := make([]authz.Scope, 0, len(permissions))
	for _, permission := range permissions {
		scopes = append(scopes, authz.Scope(permission))
	}
	_ = resolver.Tokens.Touch(ctx, tokenID)
	return authn.Principal{
		Kind:            authn.PrincipalTeamToken,
		SubjectID:       teamID,
		TeamID:          &teamID,
		TokenID:         &tokenID,
		Scopes:          authz.NewScopeSet(scopes...),
		AuthenticatedAt: resolver.now(),
	}, nil
}

func (resolver CredentialResolver) now() time.Time {
	if resolver.Now != nil {
		return resolver.Now().UTC()
	}
	return time.Now().UTC()
}

var _ authn.Resolver = CredentialResolver{}

// Keep dependency contracts close to the resolver that consumes them.
type SessionStore interface {
	GetByTokenHash(context.Context, string) (session.Record, error)
	Touch(context.Context, string) error
}

type PrincipalRepository interface {
	GetPrincipalByUserID(context.Context, string) (authn.Principal, error)
}

type TeamTokenStore interface {
	GetActiveByTokenHash(context.Context, string) (teamtoken.Token, error)
	Touch(context.Context, uuid.UUID) error
}
