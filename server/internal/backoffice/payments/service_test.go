package payments

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestListRejectsInvalidStatus(t *testing.T) {
	_, err := NewService(nil).List(context.Background(), Filter{Status: "refunded"})
	if err == nil {
		t.Fatal("expected status validation error")
	}
}

func TestReconcileRequiresReason(t *testing.T) {
	_, err := NewService(nil).Reconcile(context.Background(), uuid.NewString(), ReconcileInput{
		ProviderTransactionID: "provider-123",
		AmountUnits:           100,
		Currency:              "GHS",
		ActorUserID:           uuid.NewString(),
	})
	if err == nil {
		t.Fatal("expected reason validation error")
	}
}
func TestGetRejectsInvalidID(t *testing.T) {
	_, err := NewService(nil).Get(context.Background(), "invalid")
	if err == nil {
		t.Fatal("expected ID validation error")
	}
}
