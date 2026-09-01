package subscription

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/dugble/dugble/server/internal/messaging/email/systemmail"
	"github.com/dugble/dugble/server/internal/security/authz"
)

type planSelectionStore struct {
	scheduledPlan string
	listedLimit   int32
	listedOffset  int32
}

func (store *planSelectionStore) ListNotificationRecipients(context.Context, uuid.UUID) ([]systemmail.Recipient, error) {
	return nil, nil
}

func (store *planSelectionStore) ListCharges(_ context.Context, _ uuid.UUID, limit, offset int32) ([]Charge, error) {
	store.listedLimit, store.listedOffset = limit, offset
	return []Charge{{ID: "charge-id"}}, nil
}

func (store *planSelectionStore) GetSubscription(context.Context, uuid.UUID) (Subscription, error) {
	return Subscription{}, nil
}

func (store *planSelectionStore) SchedulePlanChange(_ context.Context, _ uuid.UUID, plan string) (Subscription, error) {
	store.scheduledPlan = plan
	return Subscription{PlanCode: "growth", PendingPlanCode: &plan}, nil
}

func (store *planSelectionStore) Cancel(context.Context, uuid.UUID) (Subscription, error) {
	return Subscription{PlanCode: "growth", CancelAtPeriodEnd: true}, nil
}

func (store *planSelectionStore) Reactivate(context.Context, uuid.UUID) (Subscription, error) {
	return Subscription{PlanCode: "growth"}, nil
}

func (store *planSelectionStore) CancelPlanChange(context.Context, uuid.UUID) (Subscription, error) {
	return Subscription{PlanCode: "growth"}, nil
}

func TestSelectPlanNormalizesPlan(t *testing.T) {
	t.Parallel()

	store := &planSelectionStore{}
	service := NewService(store)
	ctx := ownerContext()

	result, err := service.SelectPlan(ctx, SelectPlanInput{Plan: "  SCALE  "})
	if err != nil {
		t.Fatalf("select plan: %v", err)
	}
	if store.scheduledPlan != "scale" {
		t.Fatalf("scheduled plan = %q, want scale", store.scheduledPlan)
	}
	if result.PendingPlanCode == nil || *result.PendingPlanCode != "scale" {
		t.Fatalf("pending plan = %v, want scale", result.PendingPlanCode)
	}
}

func TestListChargesAppliesPaginationDefaults(t *testing.T) {
	t.Parallel()
	store := &planSelectionStore{}
	page, err := NewService(store).ListCharges(ownerContext(), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if store.listedLimit != 50 || store.listedOffset != 0 || len(page.Charges) != 1 {
		t.Fatalf("page = %+v, limit = %d, offset = %d", page, store.listedLimit, store.listedOffset)
	}
}

func TestSelectPlanRejectsInvalidPlan(t *testing.T) {
	t.Parallel()

	store := &planSelectionStore{}
	service := NewService(store)

	if _, err := service.SelectPlan(ownerContext(), SelectPlanInput{Plan: "premium"}); err == nil {
		t.Fatal("expected invalid plan error")
	}
	if store.scheduledPlan != "" {
		t.Fatalf("unexpected scheduled plan %q", store.scheduledPlan)
	}
}

func TestSelectPlanRequiresTeamUpdatePermission(t *testing.T) {
	t.Parallel()

	store := &planSelectionStore{}
	service := NewService(store)
	teamID := uuid.New()
	userID := uuid.New()
	ctx := authz.ContextWithAccess(context.Background(), authz.Access{
		Actor: authz.Actor{Type: authz.ActorTypeUser, UserID: userID},
		Scope: authz.TeamScope{TeamID: teamID, Role: string(authz.RoleMember), Status: "active"},
	})

	if _, err := service.SelectPlan(ctx, SelectPlanInput{Plan: "scale"}); err == nil {
		t.Fatal("expected permission error")
	}
	if store.scheduledPlan != "" {
		t.Fatalf("unexpected scheduled plan %q", store.scheduledPlan)
	}
}

func ownerContext() context.Context {
	return authz.ContextWithAccess(context.Background(), authz.Access{
		Actor: authz.Actor{Type: authz.ActorTypeUser, UserID: uuid.New()},
		Scope: authz.TeamScope{TeamID: uuid.New(), Role: string(authz.RoleOwner), Status: "active"},
	})
}
