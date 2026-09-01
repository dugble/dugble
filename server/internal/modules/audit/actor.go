package audit

import (
	"github.com/google/uuid"

	"github.com/dugble/dugble/server/internal/security/authz"
)

// Actor is the normalized identity attached to an audit entry.
type Actor struct {
	Type      string
	UserID    uuid.UUID
	SessionID string
	TokenID   uuid.UUID
}

func actorFromAccess(access authz.Access) Actor {
	return Actor{
		Type:      string(access.Actor.Type),
		UserID:    access.Actor.UserID,
		SessionID: access.Actor.SessionID,
		TokenID:   access.Actor.TokenID,
	}
}

func identityActor(userID uuid.UUID) Actor {
	return Actor{Type: "user", UserID: userID}
}

func (actor Actor) apply(entry *Entry) {
	if entry == nil {
		return
	}
	entry.ActorType = actor.Type
	entry.ActorUserID = actor.UserID
	entry.ActorSessionID = actor.SessionID
	entry.ActorTokenID = actor.TokenID
}

func (actor Actor) attributes() []any {
	attributes := []any{"actor_type", actor.Type}
	if actor.UserID != uuid.Nil {
		attributes = append(attributes, "actor_user_id", actor.UserID.String())
	}
	if actor.SessionID != "" {
		attributes = append(attributes, "actor_session_id", actor.SessionID)
	}
	if actor.TokenID != uuid.Nil {
		attributes = append(attributes, "actor_token_id", actor.TokenID.String())
	}
	return attributes
}
