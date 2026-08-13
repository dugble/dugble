package markets

import (
	"context"
	"testing"
)

func TestNormalizeMarketCode(t *testing.T) {
	t.Parallel()

	code, err := normalizeMarketCode(" gh ")
	if err != nil {
		t.Fatalf("normalize billing market code: %v", err)
	}
	if code != "GH" {
		t.Fatalf("expected GH, got %q", code)
	}
}

func TestCreateRejectsInvalidCurrency(t *testing.T) {
	t.Parallel()

	service := NewService(&marketFake{})
	if _, err := service.Create(context.Background(), CreateInput{Code: "GH", Currency: "cedis"}); err == nil {
		t.Fatal("expected invalid currency error")
	}
}

type marketFake struct {
	items           []Market
	currencyEnabled bool
	limit, offset   int32
}

func (r *marketFake) List(_ context.Context, limit, offset int32) ([]Market, error) {
	r.limit, r.offset = limit, offset
	return r.items, nil
}
func (*marketFake) Get(_ context.Context, code string) (Market, error) {
	return Market{Code: code, Currency: "GHS"}, nil
}
func (r *marketFake) GetCurrency(context.Context, string) (bool, error) {
	return r.currencyEnabled, nil
}
func (*marketFake) Create(_ context.Context, input CreateInput) (Market, error) {
	return Market{Code: input.Code, Currency: input.Currency}, nil
}
func (*marketFake) SetEnabled(_ context.Context, code string, enabled bool) (Market, error) {
	return Market{Code: code, IsEnabled: enabled}, nil
}
func TestListPaginates(t *testing.T) {
	r := &marketFake{items: []Market{{Code: "GH"}, {Code: "NG"}, {Code: "US"}}}
	page, err := NewService(r).List(context.Background(), ListInput{Limit: 2})
	if err != nil || r.limit != 3 || len(page.Data) != 2 || !page.HasMore {
		t.Fatalf("List() = %#v, %v", page, err)
	}
}
func TestCreateRequiresEnabledCurrencyForEnabledMarket(t *testing.T) {
	if _, err := NewService(&marketFake{}).Create(context.Background(), CreateInput{Code: "GH", Currency: "GHS"}); err == nil {
		t.Fatal("Create() expected error")
	}
	disabled := false
	item, err := NewService(&marketFake{}).Create(context.Background(), CreateInput{Code: "gh", Currency: "ghs", IsEnabled: &disabled})
	if err != nil || item.Code != "GH" || item.Currency != "GHS" {
		t.Fatalf("Create() = %#v, %v", item, err)
	}
}
func TestUpdateRequiresValueAndDisableReason(t *testing.T) {
	disabled := false
	service := NewService(&marketFake{currencyEnabled: true})
	for _, input := range []UpdateInput{{}, {IsEnabled: &disabled}} {
		if _, err := service.Update(context.Background(), "GH", input); err == nil {
			t.Fatalf("Update(%#v) expected error", input)
		}
	}
}
