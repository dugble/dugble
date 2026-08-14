package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateCreateUsesSupportedConfiguration(t *testing.T) {
	t.Parallel()

	name, region, configuration, err := validateCreate(CreateRequest{
		Name:   "Example.COM",
		Region: "eu-west-1",
		TLS:    "enforced",
	})
	if err != nil {
		t.Fatalf("validateCreate() error = %v", err)
	}
	if name != "example.com" {
		t.Fatalf("name = %q, want example.com", name)
	}
	if region != "eu-west-1" {
		t.Fatalf("region = %q, want eu-west-1", region)
	}
	if configuration.TLS != "enforced" {
		t.Fatalf("TLS = %q, want enforced", configuration.TLS)
	}
	if configuration.CustomReturnPath != DefaultCustomReturnPath {
		t.Fatalf("CustomReturnPath = %q, want %q", configuration.CustomReturnPath, DefaultCustomReturnPath)
	}
}

func TestValidateCreateDefaultsTLS(t *testing.T) {
	t.Parallel()

	_, _, configuration, err := validateCreate(CreateRequest{
		Name:   "example.com",
		Region: "eu-west-1",
	})
	if err != nil {
		t.Fatalf("validateCreate() error = %v", err)
	}
	if configuration.TLS != DefaultTLSMode {
		t.Fatalf("TLS = %q, want %q", configuration.TLS, DefaultTLSMode)
	}
}

func TestSenderDomainJSONContainsOnlySupportedConfiguration(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal(SenderDomain{
		Domain:           "example.com",
		ProviderRegion:   "eu-west-1",
		TLS:              "enforced",
		CustomReturnPath: "send",
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	body := string(payload)
	for _, field := range []string{"open_tracking", "click_tracking", "tracking_subdomain", "active_tracking_subdomain", "capabilities", "custom_return_path"} {
		if strings.Contains(body, field) {
			t.Fatalf("response contains removed field %q: %s", field, body)
		}
	}
	for _, field := range []string{"\"name\":\"example.com\"", "\"region\":\"eu-west-1\"", "\"tls\":\"enforced\""} {
		if !strings.Contains(body, field) {
			t.Fatalf("response missing %s: %s", field, body)
		}
	}
}
