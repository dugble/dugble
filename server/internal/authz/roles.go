package authz

import "fmt"

type Role string

const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
)

var ownerPermissions = func() PermissionSet {
	set := NewPermissionSet(allPermissions...)
	delete(set, PermissionTeamMemberLeave)
	return set
}()

var adminPermissions = func() PermissionSet {
	set := NewPermissionSet(allPermissions...)
	delete(set, PermissionTeamDelete)
	delete(set, PermissionTeamMemberRole)
	return set
}()

var memberPermissions = NewPermissionSet(
	PermissionTeamRead,
	PermissionTeamMembersRead,
	PermissionTeamMemberLeave,
	PermissionSenderIDsRead,
	PermissionSenderDomainsRead,
	PermissionSMSRead,
	PermissionEmailRead,
	PermissionVerifyRead,
	PermissionWebhooksRead,
	PermissionTopicsRead,
	PermissionSuppressionsRead,
	PermissionBroadcastsRead,
	PermissionBroadcastsWrite,
	PermissionSegmentsRead,
	PermissionTemplatesRead,
	PermissionContactsRead,
	PermissionContactPropertiesRead,
)

var rolePermissions = map[Role]PermissionSet{
	RoleOwner:  ownerPermissions,
	RoleAdmin:  adminPermissions,
	RoleMember: memberPermissions,
}

func IsValidRole(role Role) bool { _, ok := rolePermissions[role]; return ok }

func PermissionsForRole(role Role) (PermissionSet, error) {
	permissions, ok := rolePermissions[role]
	if !ok {
		return nil, fmt.Errorf("unsupported team role %q", role)
	}
	return permissions.Clone(), nil
}

func RoleAllows(role Role, permission Permission) bool {
	permissions, ok := rolePermissions[role]
	return ok && permissions.Has(permission)
}
