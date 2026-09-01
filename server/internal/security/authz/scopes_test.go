package authz

import "testing"

func TestNormalizeScopes(t *testing.T) {
	t.Parallel()

	normalized, err := NormalizeScopes([]Scope{Scope(PermissionSMSRead), Scope(PermissionContactsRead), Scope(PermissionSMSRead)})
	if err != nil {
		t.Fatalf("NormalizeScopes() error = %v", err)
	}
	if len(normalized) != 2 || normalized[0] != Scope(PermissionContactsRead) || normalized[1] != Scope(PermissionSMSRead) {
		t.Fatalf("NormalizeScopes() = %v", normalized)
	}
	if _, err := NormalizeScopes([]Scope{"unknown:scope"}); err == nil {
		t.Fatal("NormalizeScopes() accepted unsupported scope")
	}
}

func TestPermissionSetsAreDefensiveCopies(t *testing.T) {
	t.Parallel()

	permissions, err := PermissionsForRole(RoleMember)
	if err != nil {
		t.Fatalf("PermissionsForRole() error = %v", err)
	}
	permissions[PermissionContactsWrite] = struct{}{}
	if RoleAllows(RoleMember, PermissionContactsWrite) {
		t.Fatal("mutating returned permission set changed role policy")
	}
}
