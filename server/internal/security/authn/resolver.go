package authn

import (
	"context"
	"errors"
)

var ErrUnauthenticated = errors.New("authentication required")

// Resolver turns supported request credentials into one normalized principal.
// Implementations must document credential precedence and return
// ErrUnauthenticated when no credential can be authenticated.
type Resolver interface {
	Resolve(context.Context, Credential) (Principal, error)
}
