package ses

import (
	"slices"
	"testing"
)

func TestNormalizeSESRegion(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		supported bool
	}{
		{name: "normalizes case and space", input: "  EU-NORTH-1 ", want: "eu-north-1", supported: true},
		{name: "supports us east", input: "us-east-1", want: "us-east-1", supported: true},
		{name: "rejects unsupported region", input: "eu-west-1", want: "eu-west-1", supported: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, supported := NormalizeSESRegion(test.input)
			if got != test.want {
				t.Fatalf("expected region %q, got %q", test.want, got)
			}
			if supported != test.supported {
				t.Fatalf("expected supported=%v, got %v", test.supported, supported)
			}
		})
	}
}

func TestSupportedSESRegionsReturnsSortedCopy(t *testing.T) {
	regions := SupportedSESRegions()
	want := []string{"eu-north-1", "us-east-1"}
	if !slices.Equal(regions, want) {
		t.Fatalf("expected regions %v, got %v", want, regions)
	}

	regions[0] = "mutated"
	again := SupportedSESRegions()
	if !slices.Equal(again, want) {
		t.Fatalf("expected immutable region policy %v, got %v", want, again)
	}
}
