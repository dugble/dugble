package fxrates

import (
	"context"
	"testing"
	"time"
)

func TestCreateRejectsSameCurrencyPair(t *testing.T) {
	t.Parallel()
	s := NewService(&Repository{})
	_, err := s.Create(context.Background(), CreateInput{BaseCurrency: "USD", QuoteCurrency: "usd", Rate: "1.25", EffectiveFrom: time.Now().UTC()})
	if err == nil {
		t.Fatal("expected same-currency pair error")
	}
}

func TestCreateRejectsNonPositiveRate(t *testing.T) {
	t.Parallel()
	s := NewService(&Repository{})
	_, err := s.Create(context.Background(), CreateInput{BaseCurrency: "USD", QuoteCurrency: "GHS", Rate: "0", EffectiveFrom: time.Now().UTC()})
	if err == nil {
		t.Fatal("expected non-positive rate error")
	}
}

func TestReplaceRejectsInvalidID(t *testing.T) {
	t.Parallel()
	s := NewService(&Repository{})
	_, err := s.Replace(context.Background(), "not-a-uuid", ReplaceInput{Rate: "1.2", EffectiveFrom: time.Now().UTC(), Reason: "monthly update"})
	if err == nil {
		t.Fatal("expected invalid ID error")
	}
}
