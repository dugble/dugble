package audit

import (
	"context"

	"github.com/google/uuid"

	sentrymonitoring "github.com/dugble/dugble/server/internal/integrations/monitoring/sentry"
	"github.com/dugble/dugble/server/internal/security/authz"
)

func Record(ctx context.Context, access authz.Access, event Event) {
	actor := actorFromAccess(access)
	entry := newEntry(event)
	entry.TeamID = access.Scope.TeamID
	actor.apply(&entry)
	persist(ctx, entry)

	attributes := actor.attributes()
	attributes = append(attributes, "team_id", access.Scope.TeamID.String())
	logEvent(ctx, event, attributes)
}

func RecordIdentity(ctx context.Context, userID uuid.UUID, event Event) {
	actor := identityActor(userID)
	entry := newEntry(event)
	actor.apply(&entry)
	persist(ctx, entry)
	logEvent(ctx, event, actor.attributes())
}

func logEvent(ctx context.Context, event Event, actorAttributes []any) {
	if ctx == nil {
		ctx = context.Background()
	}
	attributes := []any{
		"audit_action", event.Action,
		"resource_type", event.ResourceType,
		"resource_id", event.ResourceID,
		"outcome", normalizedOutcome(event.Outcome),
	}
	attributes = append(attributes, actorAttributes...)
	for key, value := range event.Metadata {
		attributes = append(attributes, key, value)
	}
	sentrymonitoring.InfoContext(ctx, "security audit event", attributes...)
}
