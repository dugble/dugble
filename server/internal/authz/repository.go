package authz

import (
	"context"

	"github.com/google/uuid"
)

const (
	StatusActive    = "active"
	StatusDisabled  = "disabled"
	StatusSuspended = "suspended"
	StatusInvited   = "invited"
)

// Membership is the authorization projection of a user's team membership.
type Membership struct {
	TeamID     uuid.UUID
	UserID     uuid.UUID
	Role       string
	Status     string
	TeamStatus string
}

func (membership Membership) Active() bool {
	return membership.Status == StatusActive && (membership.TeamStatus == "" || membership.TeamStatus == StatusActive)
}

// MembershipRepository resolves the team membership needed by policy.
type MembershipRepository interface {
	GetTenantMembership(context.Context, uuid.UUID, uuid.UUID) (Membership, error)
}
