package currencies

import (
	"context"
	"testing"
)

func TestNormalizeCode(t *testing.T) {
	t.Parallel()

	code, err := normalizeCode(" usd ")
	if err != nil {
		t.Fatalf("normalize currency code: %v", err)
	}
	if code != "USD" {
		t.Fatalf("expected USD, got %q", code)
	}
}

func TestCreateRejectsInvalidMinorUnit(t *testing.T) {
	t.Parallel()

	service := NewService(&currencyFake{})
	if _, err := service.Create(context.Background(), CreateInput{Code: "USD", MinorUnit: 7}); err == nil {
		t.Fatal("expected invalid minor unit error")
	}
}

type currencyFake struct {
	items         []Currency
	limit, offset int32
}

func (r *currencyFake) List(_ context.Context, limit, offset int32) ([]Currency, error) {
	r.limit, r.offset = limit, offset
	return r.items, nil
}
func (*currencyFake) Get(context.Context, string) (Currency, error) { return Currency{}, nil }
func (*currencyFake) Create(_ context.Context, input CreateInput) (Currency, error) {
	return Currency{Code: input.Code}, nil
}
func (*currencyFake) SetEnabled(_ context.Context, code string, enabled bool) (Currency, error) {
	return Currency{Code: code, IsEnabled: enabled}, nil
}
func TestListPaginates(t *testing.T) {
	r := &currencyFake{items: []Currency{{Code: "EUR"}, {Code: "GHS"}, {Code: "USD"}}}
	page, err := NewService(r).List(context.Background(), ListInput{Limit: 2, Offset: 3})
	if err != nil || r.limit != 3 || r.offset != 3 || len(page.Data) != 2 || !page.HasMore {
		t.Fatalf("List() = %#v, %v; repository=%#v", page, err, r)
	}
}
func TestUpdateRequiresValueAndDisableReason(t *testing.T) {
	disabled := false
	service := NewService(&currencyFake{})
	for _, input := range []UpdateInput{{}, {IsEnabled: &disabled}} {
		if _, err := service.Update(context.Background(), "USD", input); err == nil {
			t.Fatalf("Update(%#v) expected error", input)
		}
	}
}
