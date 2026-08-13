package authz

import (
	"fmt"
	"sort"
)

// Scope identifies a capability granted to a credential.
type Scope string

const (
	ScopeTeamRead               Scope = Scope(PermissionTeamRead)
	ScopeTeamUpdate             Scope = Scope(PermissionTeamUpdate)
	ScopeTeamDelete             Scope = Scope(PermissionTeamDelete)
	ScopeTeamMembersRead        Scope = Scope(PermissionTeamMembersRead)
	ScopeTeamMemberLeave        Scope = Scope(PermissionTeamMemberLeave)
	ScopeTeamMemberInvite       Scope = Scope(PermissionTeamMemberInvite)
	ScopeTeamMemberRemove       Scope = Scope(PermissionTeamMemberRemove)
	ScopeTeamMemberRole         Scope = Scope(PermissionTeamMemberRole)
	ScopeTeamTokensRead         Scope = Scope(PermissionTeamTokensRead)
	ScopeTeamTokensCreate       Scope = Scope(PermissionTeamTokensCreate)
	ScopeTeamTokensUpdate       Scope = Scope(PermissionTeamTokensUpdate)
	ScopeTeamTokensRevoke       Scope = Scope(PermissionTeamTokensRevoke)
	ScopeSenderIDsRead          Scope = Scope(PermissionSenderIDsRead)
	ScopeSenderIDsCreate        Scope = Scope(PermissionSenderIDsCreate)
	ScopeSenderIDsDelete        Scope = Scope(PermissionSenderIDsDelete)
	ScopeSenderDomainsRead      Scope = Scope(PermissionSenderDomainsRead)
	ScopeSenderDomainsCreate    Scope = Scope(PermissionSenderDomainsCreate)
	ScopeSenderDomainsDelete    Scope = Scope(PermissionSenderDomainsDelete)
	ScopeSMSRead                Scope = Scope(PermissionSMSRead)
	ScopeSMSSend                Scope = Scope(PermissionSMSSend)
	ScopeEmailRead              Scope = Scope(PermissionEmailRead)
	ScopeEmailSend              Scope = Scope(PermissionEmailSend)
	ScopeVerifyRead             Scope = Scope(PermissionVerifyRead)
	ScopeVerifySend             Scope = Scope(PermissionVerifySend)
	ScopeVerifyCheck            Scope = Scope(PermissionVerifyCheck)
	ScopeVerifyManage           Scope = Scope(PermissionVerifyManage)
	ScopeWebhooksRead           Scope = Scope(PermissionWebhooksRead)
	ScopeWebhooksWrite          Scope = Scope(PermissionWebhooksWrite)
	ScopeAuditEventsRead        Scope = Scope(PermissionAuditEventsRead)
	ScopeWalletRead             Scope = Scope(PermissionWalletRead)
	ScopeWalletLedgerRead       Scope = Scope(PermissionWalletLedgerRead)
	ScopeTopicsRead             Scope = Scope(PermissionTopicsRead)
	ScopeTopicsWrite            Scope = Scope(PermissionTopicsWrite)
	ScopeSuppressionsRead       Scope = Scope(PermissionSuppressionsRead)
	ScopeSuppressionsWrite      Scope = Scope(PermissionSuppressionsWrite)
	ScopeBroadcastsRead         Scope = Scope(PermissionBroadcastsRead)
	ScopeBroadcastsWrite        Scope = Scope(PermissionBroadcastsWrite)
	ScopeBroadcastsSend         Scope = Scope(PermissionBroadcastsSend)
	ScopeSegmentsRead           Scope = Scope(PermissionSegmentsRead)
	ScopeSegmentsWrite          Scope = Scope(PermissionSegmentsWrite)
	ScopeTemplatesRead          Scope = Scope(PermissionTemplatesRead)
	ScopeTemplatesWrite         Scope = Scope(PermissionTemplatesWrite)
	ScopeContactsRead           Scope = Scope(PermissionContactsRead)
	ScopeContactsWrite          Scope = Scope(PermissionContactsWrite)
	ScopeContactPropertiesRead  Scope = Scope(PermissionContactPropertiesRead)
	ScopeContactPropertiesWrite Scope = Scope(PermissionContactPropertiesWrite)
)

var supportedScopes = func() map[Scope]struct{} {
	set := make(map[Scope]struct{}, len(allPermissions))
	for _, permission := range allPermissions {
		set[Scope(permission)] = struct{}{}
	}
	return set
}()

func IsSupportedScope(scope Scope) bool { _, ok := supportedScopes[scope]; return ok }

func NormalizeScopes(scopes []Scope) ([]Scope, error) {
	unique := make(map[Scope]struct{}, len(scopes))
	for _, scope := range scopes {
		if !IsSupportedScope(scope) {
			return nil, fmt.Errorf("unsupported authorization scope %q", scope)
		}
		unique[scope] = struct{}{}
	}
	normalized := make([]Scope, 0, len(unique))
	for scope := range unique {
		normalized = append(normalized, scope)
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i] < normalized[j] })
	return normalized, nil
}

// FirstPartyUserSessionScopes returns all currently supported capabilities.
// Team roles still restrict these capabilities at authorization time.
func FirstPartyUserSessionScopes() ScopeSet {
	scopes := make([]Scope, 0, len(allPermissions))
	for _, permission := range allPermissions {
		scopes = append(scopes, Scope(permission))
	}
	return NewScopeSet(scopes...)
}
