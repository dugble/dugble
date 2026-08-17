package relay

import (
	"errors"
	"reflect"
	"testing"
)

func TestRouteTableProviderNamesOrdersByCountryPriority(t *testing.T) {
	table, err := NewRouteTable([]Route{
		{Provider: "sendexa", CountryCode: "gh", Priority: 3, Enabled: true},
		{Provider: "moolre", CountryCode: "GH", Priority: 2, Enabled: true},
		{Provider: "mnotify", CountryCode: "GH", Priority: 1, Enabled: true},
		{Provider: "disabled", CountryCode: "GH", Priority: 4, Enabled: false},
		{Provider: "kenya", CountryCode: "KE", Priority: 1, Enabled: true},
	})
	if err != nil {
		t.Fatalf("NewRouteTable() error = %v", err)
	}

	got := table.ProviderNames(" gh ")
	want := []string{"mnotify", "moolre", "sendexa"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ProviderNames() = %v, want %v", got, want)
	}
}

func TestRouteTableRejectsDuplicateCountryPriority(t *testing.T) {
	_, err := NewRouteTable([]Route{
		{Provider: "moolre", CountryCode: "GH", Priority: 1, Enabled: true},
		{Provider: "sendexa", CountryCode: "GH", Priority: 1, Enabled: true},
	})
	if !errors.Is(err, ErrDuplicatePriority) {
		t.Fatalf("NewRouteTable() error = %v, want %v", err, ErrDuplicatePriority)
	}
}

func TestRouteTableRejectsDuplicateProviderForCountry(t *testing.T) {
	_, err := NewRouteTable([]Route{
		{Provider: "Moolre", CountryCode: "gh", Priority: 1, Enabled: true},
		{Provider: "moolre", CountryCode: "GH", Priority: 2, Enabled: false},
	})
	if !errors.Is(err, ErrDuplicateRoute) {
		t.Fatalf("NewRouteTable() error = %v, want %v", err, ErrDuplicateRoute)
	}
}
