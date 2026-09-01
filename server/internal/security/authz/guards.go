package authz

func RequireAllScopes(scopes ScopeSet, required ...Scope) error {
	if len(required) == 0 {
		return ErrInvalidPolicy
	}
	if !scopes.HasAll(required...) {
		return ErrForbidden
	}
	return nil
}
func RequireAnyScope(scopes ScopeSet, required ...Scope) error {
	if len(required) == 0 {
		return ErrInvalidPolicy
	}
	if !scopes.HasAny(required...) {
		return ErrForbidden
	}
	return nil
}
func RequireAllPermissions(permissions PermissionSet, required ...Permission) error {
	if len(required) == 0 {
		return ErrInvalidPolicy
	}
	if !permissions.HasAll(required...) {
		return ErrForbidden
	}
	return nil
}
func RequireRole(role Role, allowed ...Role) error {
	if len(allowed) == 0 {
		return ErrInvalidPolicy
	}
	for _, candidate := range allowed {
		if role == candidate {
			return nil
		}
	}
	return ErrForbidden
}
