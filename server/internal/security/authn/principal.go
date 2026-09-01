package authn

import (
	"time"

	"github.com/dugble/dugble/server/internal/security/authz"
	"github.com/google/uuid"
)

// PrincipalKind identifies the credential family that authenticated a request.
type PrincipalKind string

const (
	PrincipalUserSession PrincipalKind = "user_session"
	PrincipalTeamToken   PrincipalKind = "team_token"
)

// Principal is the transport-independent identity produced by authentication.
// The existing user-session fields remain values rather than pointers to keep
// the Phase 1 extraction behavior-compatible with current callers.
type Principal struct {
	Kind                 PrincipalKind
	SubjectID            uuid.UUID
	UserID               uuid.UUID
	SessionID            string
	TeamID               *uuid.UUID
	TokenID              *uuid.UUID
	Scopes               authz.ScopeSet
	Email                string
	Name                 string
	EmailVerified        bool
	CredentialVersion    int64
	AuthenticationMethod AuthenticationMethod
	AssuranceLevel       AssuranceLevel
	AuthenticatedAt      time.Time
	MFACompletedAt       *time.Time
}
