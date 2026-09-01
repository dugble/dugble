package authz

import "sort"

type Permission string

const (
	PermissionTeamRead               Permission = "team:read"
	PermissionTeamUpdate             Permission = "team:update"
	PermissionTeamDelete             Permission = "team:delete"
	PermissionTeamMembersRead        Permission = "team_members:read"
	PermissionTeamMemberLeave        Permission = "team_members:leave"
	PermissionTeamMemberInvite       Permission = "team_members:invite"
	PermissionTeamMemberRemove       Permission = "team_members:remove"
	PermissionTeamMemberRole         Permission = "team_members:role"
	PermissionTeamTokensRead         Permission = "team_tokens:read"
	PermissionTeamTokensCreate       Permission = "team_tokens:create"
	PermissionTeamTokensUpdate       Permission = "team_tokens:update"
	PermissionTeamTokensRevoke       Permission = "team_tokens:revoke"
	PermissionSenderIDsRead          Permission = "sender_ids:read"
	PermissionSenderIDsCreate        Permission = "sender_ids:create"
	PermissionSenderIDsDelete        Permission = "sender_ids:delete"
	PermissionSenderDomainsRead      Permission = "sender_domains:read"
	PermissionSenderDomainsCreate    Permission = "sender_domains:create"
	PermissionSenderDomainsDelete    Permission = "sender_domains:delete"
	PermissionSMSRead                Permission = "sms:read"
	PermissionSMSSend                Permission = "sms:send"
	PermissionEmailRead              Permission = "email:read"
	PermissionEmailSend              Permission = "email:send"
	PermissionVerifyRead             Permission = "verify:read"
	PermissionVerifySend             Permission = "verify:send"
	PermissionVerifyCheck            Permission = "verify:check"
	PermissionVerifyManage           Permission = "verify:manage"
	PermissionWebhooksRead           Permission = "webhooks:read"
	PermissionWebhooksWrite          Permission = "webhooks:write"
	PermissionAuditEventsRead        Permission = "audit_events:read"
	PermissionWalletRead             Permission = "wallet:read"
	PermissionWalletLedgerRead       Permission = "wallet_ledger:read"
	PermissionTopicsRead             Permission = "topics:read"
	PermissionTopicsWrite            Permission = "topics:write"
	PermissionSuppressionsRead       Permission = "suppressions:read"
	PermissionSuppressionsWrite      Permission = "suppressions:write"
	PermissionBroadcastsRead         Permission = "broadcasts:read"
	PermissionBroadcastsWrite        Permission = "broadcasts:write"
	PermissionBroadcastsSend         Permission = "broadcasts:send"
	PermissionSegmentsRead           Permission = "segments:read"
	PermissionSegmentsWrite          Permission = "segments:write"
	PermissionTemplatesRead          Permission = "templates:read"
	PermissionTemplatesWrite         Permission = "templates:write"
	PermissionContactsRead           Permission = "contacts:read"
	PermissionContactsWrite          Permission = "contacts:write"
	PermissionContactPropertiesRead  Permission = "contact_properties:read"
	PermissionContactPropertiesWrite Permission = "contact_properties:write"
)

var allPermissions = []Permission{
	PermissionTeamRead, PermissionTeamUpdate, PermissionTeamDelete, PermissionTeamMembersRead, PermissionTeamMemberLeave, PermissionTeamMemberInvite, PermissionTeamMemberRemove, PermissionTeamMemberRole,
	PermissionTeamTokensRead, PermissionTeamTokensCreate, PermissionTeamTokensUpdate, PermissionTeamTokensRevoke, PermissionSenderIDsRead, PermissionSenderIDsCreate, PermissionSenderIDsDelete,
	PermissionSenderDomainsRead, PermissionSenderDomainsCreate, PermissionSenderDomainsDelete, PermissionSMSRead, PermissionSMSSend, PermissionEmailRead, PermissionEmailSend,
	PermissionVerifyRead, PermissionVerifySend, PermissionVerifyCheck, PermissionVerifyManage, PermissionWebhooksRead, PermissionWebhooksWrite, PermissionAuditEventsRead,
	PermissionWalletRead, PermissionWalletLedgerRead, PermissionTopicsRead, PermissionTopicsWrite, PermissionSuppressionsRead, PermissionSuppressionsWrite,
	PermissionBroadcastsRead, PermissionBroadcastsWrite, PermissionBroadcastsSend, PermissionSegmentsRead, PermissionSegmentsWrite, PermissionTemplatesRead, PermissionTemplatesWrite,
	PermissionContactsRead, PermissionContactsWrite, PermissionContactPropertiesRead, PermissionContactPropertiesWrite,
}

type PermissionSet map[Permission]struct{}

func NewPermissionSet(values ...Permission) PermissionSet {
	set := make(PermissionSet, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}
func (set PermissionSet) Has(value Permission) bool { _, ok := set[value]; return ok }
func (set PermissionSet) HasAll(values ...Permission) bool {
	for _, value := range values {
		if !set.Has(value) {
			return false
		}
	}
	return true
}
func (set PermissionSet) HasAny(values ...Permission) bool {
	for _, value := range values {
		if set.Has(value) {
			return true
		}
	}
	return false
}
func (set PermissionSet) Permissions() []Permission {
	values := make([]Permission, 0, len(set))
	for value := range set {
		values = append(values, value)
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	return values
}
func (set PermissionSet) Clone() PermissionSet { return NewPermissionSet(set.Permissions()...) }

// PermissionsImpliedByScopes expands scopes into application permissions.
// Existing team-token values are intentionally one-to-one during extraction.
func PermissionsImpliedByScopes(scopes ScopeSet) PermissionSet {
	permissions := NewPermissionSet()
	for scope := range scopes {
		if IsSupportedScope(scope) {
			permissions[Permission(scope)] = struct{}{}
		}
	}
	return permissions
}
