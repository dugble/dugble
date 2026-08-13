package authz

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestAuthorizeUserRequiresScopeAndRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		scopes     ScopeSet
		role       Role
		permission Permission
		want       error
	}{
		{name: "owner with scope", scopes: NewScopeSet(Scope(PermissionTeamDelete)), role: RoleOwner, permission: PermissionTeamDelete},
		{name: "role without scope", scopes: NewScopeSet(Scope(PermissionTeamRead)), role: RoleOwner, permission: PermissionTeamDelete, want: ErrForbidden},
		{name: "scope without role", scopes: NewScopeSet(Scope(PermissionTeamDelete)), role: RoleAdmin, permission: PermissionTeamDelete, want: ErrForbidden},
		{name: "unknown role", scopes: NewScopeSet(Scope(PermissionTeamRead)), role: Role("unknown"), permission: PermissionTeamRead, want: ErrForbidden},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if err := AuthorizeUser(test.scopes, test.role, test.permission); !errors.Is(err, test.want) {
				t.Fatalf("AuthorizeUser() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestRolePolicyPreservesExistingContactPermissions(t *testing.T) {
	t.Parallel()

	if !RoleAllows(RoleAdmin, PermissionContactsWrite) || !RoleAllows(RoleMember, PermissionContactsRead) {
		t.Fatal("expected existing contact permissions to be preserved")
	}
	if RoleAllows(RoleMember, PermissionContactsWrite) {
		t.Fatal("member unexpectedly has contacts:write")
	}
}

func TestRolePolicyPreservesOwnerAndAdminInvariants(t *testing.T) {
	t.Parallel()

	if RoleAllows(RoleOwner, PermissionTeamMemberLeave) {
		t.Fatal("owner unexpectedly has team_members:leave")
	}
	if !RoleAllows(RoleOwner, PermissionTeamDelete) || !RoleAllows(RoleOwner, PermissionTeamMemberRole) {
		t.Fatal("owner is missing owner-only permissions")
	}
	if RoleAllows(RoleAdmin, PermissionTeamDelete) || RoleAllows(RoleAdmin, PermissionTeamMemberRole) {
		t.Fatal("admin unexpectedly has owner-only permissions")
	}
}

func TestRoleAuthorizer(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	user := Access{Actor: Actor{Type: ActorTypeUser, UserID: uuid.New()}, Scope: TeamScope{TeamID: teamID, Role: string(RoleMember), Status: StatusActive, Scopes: FirstPartyUserSessionScopes()}}
	if decision := Authorize(user, PermissionContactsRead); !decision.Allowed {
		t.Fatalf("Authorize() denied member read: %s", decision.Reason)
	}
	if decision := Authorize(user, PermissionContactsWrite); decision.Allowed {
		t.Fatal("Authorize() allowed member write")
	}
	user.Scope.Scopes = nil
	if decision := Authorize(user, PermissionContactsRead); !decision.Allowed {
		t.Fatalf("Authorize() changed unscoped first-party user behavior: %s", decision.Reason)
	}

	token := Access{Actor: Actor{Type: ActorTypeTeamToken, TokenID: uuid.New()}, Scope: TeamScope{TeamID: teamID, Status: StatusActive, Scopes: NewScopeSet(Scope(PermissionSMSSend))}}
	if decision := Authorize(token, PermissionSMSSend); !decision.Allowed {
		t.Fatalf("Authorize() denied scoped team token: %s", decision.Reason)
	}
	if decision := Authorize(token, PermissionSMSRead); decision.Allowed {
		t.Fatal("Authorize() allowed missing team-token scope")
	}
}

func TestAuthorizationRejectsEmptyPolicy(t *testing.T) {
	t.Parallel()
	if !errors.Is(AuthorizeUser(FirstPartyUserSessionScopes(), RoleOwner), ErrInvalidPolicy) {
		t.Fatal("AuthorizeUser() accepted empty policy")
	}
	if !errors.Is(AuthorizeTeamToken(NewScopeSet()), ErrInvalidPolicy) {
		t.Fatal("AuthorizeTeamToken() accepted empty policy")
	}
}
