package middleware

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dugble/dugble/server/internal/authn"
	"github.com/dugble/dugble/server/internal/authz"
	"github.com/dugble/dugble/server/internal/modules/session"
	"github.com/dugble/dugble/server/internal/modules/teamtoken"
)

type resolverSessionStore struct {
	record  session.Record
	getErr  error
	touched string
}

func (store *resolverSessionStore) GetByTokenHash(context.Context, string) (session.Record, error) {
	return store.record, store.getErr
}
func (store *resolverSessionStore) Touch(_ context.Context, id string) error {
	store.touched = id
	return nil
}

type resolverUsers struct{ principal authn.Principal }

func (users resolverUsers) GetPrincipalByUserID(context.Context, string) (authn.Principal, error) {
	return users.principal, nil
}

type resolverTokenStore struct {
	token   teamtoken.Token
	touched uuid.UUID
}

func (store *resolverTokenStore) GetActiveByTokenHash(context.Context, string) (teamtoken.Token, error) {
	return store.token, nil
}
func (store *resolverTokenStore) Touch(_ context.Context, id uuid.UUID) error {
	store.touched = id
	return nil
}

func TestCredentialResolverResolvesSession(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 11, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()
	sessions := &resolverSessionStore{record: session.Record{
		ID: "session-id", UserID: userID, ExpiresAt: now.Add(time.Hour),
		Authentication: session.Authentication{CredentialVersion: 4, AuthenticatedAt: now.Add(-time.Minute)},
	}}
	resolver := CredentialResolver{Sessions: sessions, Users: resolverUsers{principal: authn.Principal{UserID: userID, CredentialVersion: 4}}, Now: func() time.Time { return now }}

	principal, err := resolver.Resolve(context.Background(), authn.Credential{SessionToken: " session-secret "})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if principal.Kind != authn.PrincipalUserSession || principal.SubjectID != userID || principal.SessionID != "session-id" {
		t.Fatalf("Resolve() principal = %#v", principal)
	}
	if !principal.Scopes.Has(authz.ScopeContactsRead) || sessions.touched != "session-id" {
		t.Fatal("Resolve() did not grant first-party scopes or touch the session")
	}
}

func TestCredentialResolverBearerTakesPrecedence(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 11, 12, 0, 0, 0, time.UTC)
	teamID, tokenID := uuid.New(), uuid.New()
	tokens := &resolverTokenStore{token: teamtoken.Token{ID: tokenID.String(), TeamID: teamID.String(), Permissions: []string{string(authz.PermissionSMSSend)}}}
	resolver := CredentialResolver{Tokens: tokens, Now: func() time.Time { return now }}

	principal, err := resolver.Resolve(context.Background(), authn.Credential{BearerToken: teamtoken.TokenPrefix + "secret", SessionToken: "ignored"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if principal.Kind != authn.PrincipalTeamToken || principal.TeamID == nil || *principal.TeamID != teamID || !principal.Scopes.Has(authz.ScopeSMSSend) {
		t.Fatalf("Resolve() principal = %#v", principal)
	}
	if tokens.touched != tokenID {
		t.Fatal("Resolve() did not record team-token usage")
	}
}

func TestCredentialResolverRejectsMissingCredential(t *testing.T) {
	t.Parallel()

	_, err := (CredentialResolver{}).Resolve(context.Background(), authn.Credential{})
	if !errors.Is(err, authn.ErrUnauthenticated) {
		t.Fatalf("Resolve() error = %v, want ErrUnauthenticated", err)
	}
}
