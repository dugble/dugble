package ses

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func TestRegionalConfigOverridesRegion(t *testing.T) {
	client := &Client{awsConfig: aws.Config{Region: "us-east-1"}}
	config := client.regionalConfig("eu-north-1")

	if config.Region != "eu-north-1" {
		t.Fatalf("expected regional config eu-north-1, got %q", config.Region)
	}
	if client.awsConfig.Region != "us-east-1" {
		t.Fatalf("expected base AWS config to remain us-east-1, got %q", client.awsConfig.Region)
	}
}

func TestV2SendingClientRejectsUnsupportedRegion(t *testing.T) {
	client := &Client{
		awsConfig:        aws.Config{Region: "us-east-1"},
		v2SendingClients: make(map[string]sesV2SendAPI),
	}

	if _, err := client.v2SendingClient("eu-west-1"); err == nil {
		t.Fatal("expected unsupported SES delivery region error")
	}
}

func TestV2SendingClientCachesNormalizedRegion(t *testing.T) {
	client := &Client{
		awsConfig:        aws.Config{Region: "us-east-1"},
		v2SendingClients: make(map[string]sesV2SendAPI),
	}

	first, err := client.v2SendingClient(" EU-NORTH-1 ")
	if err != nil {
		t.Fatalf("create regional SES client: %v", err)
	}
	second, err := client.v2SendingClient("eu-north-1")
	if err != nil {
		t.Fatalf("reuse regional SES client: %v", err)
	}
	if first != second {
		t.Fatal("expected normalized region to reuse cached SES client")
	}
	if len(client.v2SendingClients) != 1 {
		t.Fatalf("expected one cached SES client, got %d", len(client.v2SendingClients))
	}
}

func TestIdentityAndTenantClientsUseDefaultRegion(t *testing.T) {
	client := &Client{
		defaultRegion:   "eu-north-1",
		awsConfig:       aws.Config{Region: "us-east-1"},
		identityClients: make(map[string]sesIdentityAPI),
		tenantClients:   make(map[string]sesTenantAPI),
	}

	if _, err := client.identityClient(""); err != nil {
		t.Fatalf("create default-region identity client: %v", err)
	}
	if _, ok := client.identityClients["eu-north-1"]; !ok {
		t.Fatal("expected identity client cached under default region")
	}

	if _, err := client.tenantClient(""); err != nil {
		t.Fatalf("create default-region tenant client: %v", err)
	}
	if _, ok := client.tenantClients["eu-north-1"]; !ok {
		t.Fatal("expected tenant client cached under default region")
	}
}
