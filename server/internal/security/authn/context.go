package authn

import "context"

type principalContextKey struct{}

// ContextWithPrincipal returns a child context containing principal.
func ContextWithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

// PrincipalFromContext returns the authenticated principal attached to ctx.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(Principal)
	return principal, ok
}

// MustPrincipalFromContext returns the authenticated principal or panics.
// Use it only after an authentication guard has guaranteed the principal.
func MustPrincipalFromContext(ctx context.Context) Principal {
	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		panic("authn: principal missing from context")
	}
	return principal
}
