package authz

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrForbidden     = errors.New("authorization denied")
	ErrInvalidPolicy = errors.New("authorization policy requires at least one permission")
)

// AuthorizeUser evaluates both credential scopes and the user's team role.
func AuthorizeUser(scopes ScopeSet, role Role, required ...Permission) error {
	if len(required) == 0 {
		return ErrInvalidPolicy
	}
	rolePermissions, err := PermissionsForRole(role)
	if err != nil || !rolePermissions.HasAll(required...) || !PermissionsImpliedByScopes(scopes).HasAll(required...) {
		return ErrForbidden
	}
	return nil
}

// AuthorizeTeamToken evaluates the token's credential scopes.
// The token's team identity must be checked before calling this function.
func AuthorizeTeamToken(scopes ScopeSet, required ...Permission) error {
	if len(required) == 0 {
		return ErrInvalidPolicy
	}
	if !PermissionsImpliedByScopes(scopes).HasAll(required...) {
		return ErrForbidden
	}
	return nil
}

type Decision struct {
	Allowed bool
	Reason  string
}
type Authorizer interface {
	Authorize(Access, Permission) Decision
}
type RoleAuthorizer struct{}

func (RoleAuthorizer) Authorize(access Access, permission Permission) Decision {
	if access.Scope.TeamID == uuid.Nil {
		return Decision{Reason: "tenant scope is missing"}
	}
	// Disabled teams are read-only for normal operations, but their owner must
	// retain the ability to delete the disabled team so account cleanup can
	// remove the membership that blocks user deletion.
	if access.Scope.Status != StatusActive && !(permission == PermissionTeamDelete && Role(access.Scope.Role) == RoleOwner) {
		return Decision{Reason: "active tenant scope is required"}
	}
	if !access.Actor.IsUser() && !access.Actor.IsTeamToken() {
		return Decision{Reason: "authenticated actor is required"}
	}
	if permission == "" {
		return Decision{Allowed: true}
	}
	scopes := access.Scope.Scopes
	if len(scopes) == 0 && len(access.Scope.Permissions) > 0 {
		scopes = NewScopeSet()
		for _, candidate := range access.Scope.Permissions {
			scopes[Scope(candidate)] = struct{}{}
		}
	}
	if access.Actor.IsUser() && len(scopes) == 0 {
		// Existing user sessions were authorized exclusively by their team role.
		// Treat an unscoped user access context as first-party during extraction.
		scopes = FirstPartyUserSessionScopes()
	}
	if access.Actor.IsTeamToken() {
		if AuthorizeTeamToken(scopes, permission) == nil {
			return Decision{Allowed: true}
		}
		return Decision{Reason: "team token permission is required"}
	}
	if AuthorizeUser(scopes, Role(access.Scope.Role), permission) == nil {
		return Decision{Allowed: true}
	}
	return Decision{Reason: "team permission is required"}
}

func DefaultAuthorizer() Authorizer { return RoleAuthorizer{} }
func Authorize(access Access, permission Permission) Decision {
	return DefaultAuthorizer().Authorize(access, permission)
}
func ResolveAccess(ctx context.Context, permission Permission) (Access, Decision) {
	access, ok := AccessFromContext(ctx)
	if !ok {
		return Access{}, Decision{Reason: "team context is required"}
	}
	return access, Authorize(access, permission)
}
