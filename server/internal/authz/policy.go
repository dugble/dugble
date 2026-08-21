package authz

import (
	"github.com/google/uuid"
)

type RoleAuthorizer struct{}

func (RoleAuthorizer) Authorize(access Access, permission Permission) Decision {
	if access.Scope.TeamID == uuid.Nil {
		return Decision{Reason: "tenant scope is missing"}
	}
	// Disabled teams are read-only for normal operations, but their owner must
	// retain the ability to delete the disabled team so account cleanup can
	// remove the membership that blocks user deletion.
	if access.Scope.Status != StatusActive && (permission != PermissionTeamDelete || Role(access.Scope.Role) != RoleOwner) {
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
		scopes = access.Scope.Permissions
	}
	if permission == PermissionTeamDelete && Role(access.Scope.Role) == RoleOwner {
		return Decision{Allowed: true}
	}
	if !hasPermission(scopes, permission) {
		return Decision{Reason: "permission denied"}
	}
	return Decision{Allowed: true}
}

func hasPermission(scopes []string, permission Permission) bool {
	for _, scope := range scopes {
		if Permission(scope) == permission {
			return true
		}
	}
	return false
}
