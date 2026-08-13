package subscription

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type chargerStub struct{ result Result }

func (s chargerStub) ChargePeriod(context.Context, pgx.Tx, Input) (Result, error) {
	return s.result, nil
}
func TestChargePeriodRejectsInvalidInput(t *testing.T) {
	service := NewService(nil)
	_, err := service.ChargePeriod(context.Background(), nil, Input{PeriodStart: time.Now(), PeriodEnd: time.Now()})
	if !errors.Is(err, ErrInvalidPeriod) {
		t.Fatalf("error = %v", err)
	}
}
func TestChargePeriodReturnsConcreteResult(t *testing.T) {
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	expected := Result{Outcome: OutcomeApplied, Status: "applied", AmountUnits: 100}
	service := NewService(chargerStub{expected})
	result, err := service.ChargePeriod(context.Background(), nil, Input{SubscriptionID: uuid.New(), TeamID: uuid.New(), PlanCode: "growth", PeriodStart: start, PeriodEnd: start.AddDate(0, 1, 0)})
	if err != nil {
		t.Fatal(err)
	}
	if result != expected {
		t.Fatalf("%+v", result)
	}
}
