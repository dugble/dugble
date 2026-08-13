package sms_test

import (
	"errors"
	"testing"

	platformsms "github.com/coffeyvidzro/dugble/server/internal/platform/sms"
)

func TestSupportedDestinationCountryUsesResolutionCatalog(t *testing.T) {
	t.Parallel()

	if !platformsms.IsSupportedDestinationCountry("gh") {
		t.Fatal("IsSupportedDestinationCountry(gh) = false")
	}
	if !platformsms.IsSupportedDestinationCountry("ke") {
		t.Fatal("IsSupportedDestinationCountry(ke) = false")
	}
	if !platformsms.IsSupportedDestinationCountry("ng") {
		t.Fatal("IsSupportedDestinationCountry(ng) = false")
	}

	country, err := platformsms.ResolveDestinationCountry("+254700000001")
	if err != nil || country != platformsms.CountryKenya {
		t.Fatalf("ResolveDestinationCountry(KE) = %q, %v", country, err)
	}
	country, err = platformsms.ResolveDestinationCountry("+2348000000000")
	if err != nil || country != platformsms.CountryNigeria {
		t.Fatalf("ResolveDestinationCountry(NG) = %q, %v", country, err)
	}
	if _, err := platformsms.ResolveDestinationCountry("+27710000000"); !errors.Is(err, platformsms.ErrUnsupportedDestination) {
		t.Fatalf("ResolveDestinationCountry(ZA) error = %v", err)
	}

	destinations := platformsms.SupportedDestinations()
	if len(destinations) != 3 {
		t.Fatalf("SupportedDestinations() count = %d, want 3", len(destinations))
	}
	destinations[0].CountryCode = "XX"
	if !platformsms.IsSupportedDestinationCountry(platformsms.CountryGhana) {
		t.Fatal("SupportedDestinations() exposed mutable internal state")
	}
}
