package wallets

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestValidatePageUsesDefaults(t *testing.T) {
	t.Parallel()

	limit, offset, err := validatePage(0, 0)
	if err != nil {
		t.Fatalf("validatePage() error = %v", err)
	}
	if limit != defaultPageLimit || offset != 0 {
		t.Fatalf("validatePage() = %d, %d", limit, offset)
	}
}

func TestListTransactionsRejectsInvalidTeamID(t *testing.T) {
	t.Parallel()

	_, err := NewService(&Repository{}).ListTransactions(
		context.Background(),
		TransactionListInput{TeamID: "invalid"},
	)
	if err == nil {
		t.Fatal("ListTransactions() expected an error")
	}
}

func TestAdjustRejectsZeroAmount(t *testing.T) {
	t.Parallel()

	_, err := NewService(&Repository{}).Adjust(
		context.Background(),
		uuid.NewString(),
		AdjustmentInput{ReferenceID: "ticket"},
	)
	if err == nil {
		t.Fatal("Adjust() expected an error")
	}
}

func TestAdjustRequiresReason(t *testing.T) {
	t.Parallel()

	_, err := NewService(&Repository{}).Adjust(
		context.Background(),
		uuid.NewString(),
		AdjustmentInput{AmountUnits: 100, ReferenceID: "ticket", ActorUserID: uuid.NewString()},
	)
	if err == nil {
		t.Fatal("Adjust() expected a reason validation error")
	}
}
