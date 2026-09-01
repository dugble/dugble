package authz

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type ActorType string

const (
	ActorTypeUser      ActorType = "user"
	ActorTypeTeamToken ActorType = "team_token"
)

type Actor struct {
	Type      ActorType
	UserID    uuid.UUID
	SessionID string
	TokenID   uuid.UUID
}

func (actor Actor) IsUser() bool { return actor.Type == ActorTypeUser && actor.UserID != uuid.Nil }
func (actor Actor) IsTeamToken() bool {
	return actor.Type == ActorTypeTeamToken && actor.TokenID != uuid.Nil
}

// UserIDPtr returns the authenticated user's ID when the actor is a user.
// Team-token actors intentionally return nil so nullable user attribution
// columns do not receive uuid.Nil and violate their users foreign key.
func (actor Actor) UserIDPtr() *uuid.UUID {
	if !actor.IsUser() {
		return nil
	}
	id := actor.UserID
	return &id
}

type TeamScope struct {
	TeamID      uuid.UUID
	Role        string
	Status      string
	Scopes      ScopeSet
	Permissions []Permission
}
type Access struct {
	Actor Actor
	Scope TeamScope
}

// AccessibleTeamID is a team identifier derived from an established access
// boundary. Repository entry points use it to avoid accepting arbitrary UUIDs.
type AccessibleTeamID struct{ value uuid.UUID }

var ErrInvalidAccessibleTeamID = errors.New("accessible team scope is required")

func (id AccessibleTeamID) UUID() (uuid.UUID, error) {
	if !id.Valid() {
		return uuid.Nil, ErrInvalidAccessibleTeamID
	}
	return id.value, nil
}

func (id AccessibleTeamID) Valid() bool { return id.value != uuid.Nil }

// AuthorizedTeamID returns a repository-safe team identifier. Callers should
// obtain Access through ResolveAccess before invoking this method.
func (access Access) AuthorizedTeamID() AccessibleTeamID {
	if access.Scope.TeamID == uuid.Nil || access.Scope.Status != StatusActive || (!access.Actor.IsUser() && !access.Actor.IsTeamToken()) {
		return AccessibleTeamID{}
	}
	return AccessibleTeamID{value: access.Scope.TeamID}
}

type accessContextKey struct{}

func ContextWithAccess(ctx context.Context, access Access) context.Context {
	return context.WithValue(ctx, accessContextKey{}, access)
}
func AccessFromContext(ctx context.Context) (Access, bool) {
	access, ok := ctx.Value(accessContextKey{}).(Access)
	return access, ok
}
