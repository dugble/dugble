package payment

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/dugble/dugble/server/internal/messaging/email/systemmail"
)

type storeStub struct {
	transaction Transaction
	completed   bool
	failed      bool
	recipients  []Recipient
}

func (s *storeStub) MarkFailed(_ context.Context, transaction Transaction) (Transaction, error) {
	s.failed = true
	transaction.Status = StatusFailed
	return transaction, nil
}
func (s *storeStub) ListRecipients(context.Context, uuid.UUID) ([]Recipient, error) {
	return s.recipients, nil
}

type notifierStub struct {
	inputs []systemmail.SendWalletTopUpResultInput
}

func (s *notifierStub) SendWalletTopUpResult(_ context.Context, input systemmail.SendWalletTopUpResultInput) error {
	s.inputs = append(s.inputs, input)
	return nil
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

func TestCompleteNotifiesOwners(t *testing.T) {
	store := &storeStub{
		transaction: Transaction{ID: uuid.NewString(), TeamID: uuid.NewString(), AmountUnits: 2500, Currency: CurrencyGHS, ClientReference: "ref", Status: StatusPending},
		recipients:  []Recipient{{Name: "Ada", Email: "ada@example.com", TeamName: "Acme"}},
	}
	notifier := &notifierStub{}
	transaction, err := NewService(store).WithNotifier(notifier).Complete(context.Background(), CompleteInput{Provider: ProviderHubtel, ClientReference: "ref", ProviderTransactionID: "txn", AmountUnits: 2500})
	if err != nil {
		t.Fatal(err)
	}
	if transaction.Status != StatusPaid || len(notifier.inputs) != 1 || notifier.inputs[0].Status != StatusPaid {
		t.Fatalf("transaction=%+v notifications=%+v", transaction, notifier.inputs)
	}
}

func TestFailIsIdempotentAndNotifiesOnlyOnTransition(t *testing.T) {
	store := &storeStub{
		transaction: Transaction{ID: uuid.NewString(), TeamID: uuid.NewString(), AmountUnits: 2500, Currency: CurrencyGHS, ClientReference: "ref", Status: StatusPending},
		recipients:  []Recipient{{Email: "owner@example.com"}},
	}
	notifier := &notifierStub{}
	service := NewService(store).WithNotifier(notifier)
	transaction, err := service.Fail(context.Background(), FailInput{Provider: ProviderHubtel, ClientReference: "ref"})
	if err != nil {
		t.Fatal(err)
	}
	if transaction.Status != StatusFailed || !store.failed || len(notifier.inputs) != 1 {
		t.Fatalf("transaction=%+v failed=%v notifications=%d", transaction, store.failed, len(notifier.inputs))
	}
	store.transaction = transaction
	if _, err := service.Fail(context.Background(), FailInput{Provider: ProviderHubtel, ClientReference: "ref"}); err != nil {
		t.Fatal(err)
	}
	if len(notifier.inputs) != 1 {
		t.Fatalf("notifications=%d, want 1", len(notifier.inputs))
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
