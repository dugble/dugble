package routing_test

import (
	"errors"
	"testing"

	platformsms "github.com/dugble/dugble/server/internal/platform/sms"
	"github.com/dugble/dugble/server/internal/platform/sms/routing"
)

func TestDefaultConfigPrioritizesCountryProviders(t *testing.T) {
	t.Parallel()

	config := routing.DefaultConfig()
	var ghana, kenya []routing.Route
	for _, route := range config.Routes {
		if !route.Enabled {
			continue
		}
		switch route.DestinationCountry {
		case platformsms.CountryGhana:
			ghana = append(ghana, route)
		case platformsms.CountryKenya:
			kenya = append(kenya, route)
		}
	}
	if len(ghana) != 2 || ghana[0].ProviderID != "mnotify" || ghana[0].Priority != 1 || ghana[1].ProviderID != "moolre" || ghana[1].Priority != 2 {
		t.Fatalf("Ghana routes = %#v", ghana)
	}
	if len(kenya) != 2 || kenya[0].ProviderID != "leamout" || kenya[0].Priority != 1 || kenya[1].ProviderID != "runnage" || kenya[1].Priority != 2 {
		t.Fatalf("Kenya routes = %#v", kenya)
	}
}

func TestConfigRejectsInvalidRoutes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		routes []routing.Route
		target error
	}{
		{
			name: "invalid priority",
			routes: []routing.Route{
				{ProviderID: "leamout", DestinationCountry: platformsms.CountryKenya, Priority: 0, Enabled: true},
			},
			target: routing.ErrInvalidPriority,
		},
		{
			name: "duplicate provider",
			routes: []routing.Route{
				{ProviderID: "leamout", DestinationCountry: platformsms.CountryKenya, Priority: 1, Enabled: true},
				{ProviderID: "leamout", DestinationCountry: platformsms.CountryKenya, Priority: 2, Enabled: false},
			},
			target: routing.ErrDuplicateProvider,
		},
		{
			name: "duplicate priority",
			routes: []routing.Route{
				{ProviderID: "leamout", DestinationCountry: platformsms.CountryKenya, Priority: 1, Enabled: true},
				{ProviderID: "runnage", DestinationCountry: platformsms.CountryKenya, Priority: 1, Enabled: true},
			},
			target: routing.ErrDuplicatePriority,
		},
		{
			name: "malformed country",
			routes: []routing.Route{
				{ProviderID: "leamout", DestinationCountry: "KEN", Priority: 1, Enabled: true},
			},
			target: routing.ErrInvalidCountryCode,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := (routing.Config{Routes: test.routes}).Validate()
			if !errors.Is(err, test.target) {
				t.Fatalf("Validate() error = %v, want %v", err, test.target)
			}
		})
	}
}
