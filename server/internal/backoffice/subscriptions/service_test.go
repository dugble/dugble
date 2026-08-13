package subscriptions

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestListRejectsInvalidStatus(t *testing.T) {
	_, err := NewService(nil).List(context.Background(), Filter{Status: "paused"})
	if err == nil {
		t.Fatal("expected status validation error")
	}
}

func TestChangePlanRequiresPlanCode(t *testing.T) {
	_, err := NewService(nil).ChangePlan(context.Background(), uuid.NewString(), ChangePlanInput{})
	if err == nil {
		t.Fatal("expected plan code validation error")
	}
}

func TestCancelRequiresReason(t *testing.T) {
	_, err := NewService(nil).Cancel(context.Background(), uuid.NewString(), ActionInput{ActorUserID: uuid.NewString()})
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
