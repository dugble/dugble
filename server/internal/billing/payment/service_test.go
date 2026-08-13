package payment

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type storeStub struct {
	transaction Transaction
	completed   bool
}

func (s *storeStub) Create(context.Context, uuid.UUID, CreateInput) (Transaction, error) {
	return s.transaction, nil
}
func (s *storeStub) GetByClientReference(context.Context, string, string) (Transaction, error) {
	return s.transaction, nil
}
func (s *storeStub) MarkPaidAndCredit(_ context.Context, transaction Transaction, id string) (Transaction, error) {
	s.completed = true
	transaction.Status = StatusPaid
	transaction.ProviderTransactionID = &id
	return transaction, nil
}

func TestCompleteRejectsAmountMismatch(t *testing.T) {
	store := &storeStub{transaction: Transaction{AmountUnits: 100, Status: StatusPending}}
	_, err := NewService(store).Complete(context.Background(), CompleteInput{Provider: ProviderHubtel, ClientReference: "ref", ProviderTransactionID: "txn", AmountUnits: 99})
	if err == nil || store.completed {
		t.Fatalf("expected mismatch error, got %v", err)
	}
}

func TestCompleteIsIdempotent(t *testing.T) {
	id := "txn"
	store := &storeStub{transaction: Transaction{AmountUnits: 100, Status: StatusPaid, ProviderTransactionID: &id}}
	got, err := NewService(store).Complete(context.Background(), CompleteInput{Provider: ProviderHubtel, ClientReference: "ref", ProviderTransactionID: id, AmountUnits: 100})
	if err != nil || got.Status != StatusPaid || store.completed {
		t.Fatalf("result=%+v completed=%v err=%v", got, store.completed, err)
	}
}

func TestCreateValidatesInput(t *testing.T) {
	_, err := NewService(&storeStub{}).Create(context.Background(), CreateInput{TeamID: "bad", Provider: ProviderHubtel, ClientReference: "ref", Currency: CurrencyGHS, AmountUnits: 100})
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestCreateRejectsInvalidCurrency(t *testing.T) {
	_, err := NewService(&storeStub{}).Create(context.Background(), CreateInput{TeamID: uuid.NewString(), Provider: ProviderHubtel, ClientReference: "ref", Currency: "GH", AmountUnits: 100})
	if err == nil {
		t.Fatalf("expected currency validation error")
	}
}
