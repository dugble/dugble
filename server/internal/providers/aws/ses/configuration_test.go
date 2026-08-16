package ses

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/smithy-go"
)

type configurationClientStub struct {
	createConfigurationSets []string
	createDestination       *sesv2.CreateConfigurationSetEventDestinationInput
	updateDestination       *sesv2.UpdateConfigurationSetEventDestinationInput
}

func (stub *configurationClientStub) CreateConfigurationSet(_ context.Context, input *sesv2.CreateConfigurationSetInput, _ ...func(*sesv2.Options)) (*sesv2.CreateConfigurationSetOutput, error) {
	stub.createConfigurationSets = append(stub.createConfigurationSets, stringValue(input.ConfigurationSetName))
	return nil, &smithy.GenericAPIError{Code: "AlreadyExistsException", Message: "configuration set already exists", Fault: smithy.FaultClient}
}

func (stub *configurationClientStub) DeleteConfigurationSet(context.Context, *sesv2.DeleteConfigurationSetInput, ...func(*sesv2.Options)) (*sesv2.DeleteConfigurationSetOutput, error) {
	return &sesv2.DeleteConfigurationSetOutput{}, nil
}

func (stub *configurationClientStub) CreateConfigurationSetEventDestination(_ context.Context, input *sesv2.CreateConfigurationSetEventDestinationInput, _ ...func(*sesv2.Options)) (*sesv2.CreateConfigurationSetEventDestinationOutput, error) {
	stub.createDestination = input
	return nil, &smithy.GenericAPIError{Code: "AlreadyExistsException", Message: "event destination already exists", Fault: smithy.FaultClient}
}

func (stub *configurationClientStub) UpdateConfigurationSetEventDestination(_ context.Context, input *sesv2.UpdateConfigurationSetEventDestinationInput, _ ...func(*sesv2.Options)) (*sesv2.UpdateConfigurationSetEventDestinationOutput, error) {
	stub.updateDestination = input
	return &sesv2.UpdateConfigurationSetEventDestinationOutput{}, nil
}

func (stub *configurationClientStub) DeleteConfigurationSetEventDestination(context.Context, *sesv2.DeleteConfigurationSetEventDestinationInput, ...func(*sesv2.Options)) (*sesv2.DeleteConfigurationSetEventDestinationOutput, error) {
	return &sesv2.DeleteConfigurationSetEventDestinationOutput{}, nil
}

func TestEnsureSharedConfigurationSetsConvergesExistingSets(t *testing.T) {
	stub := &configurationClientStub{}
	client := &Client{
		defaultRegion:        "eu-north-1",
		configurationClients: map[string]sesConfigurationAPI{"eu-north-1": stub},
	}

	if err := client.EnsureSharedConfigurationSets(context.Background(), "eu-north-1"); err != nil {
		t.Fatalf("ensure shared SES configuration sets: %v", err)
	}
	if len(stub.createConfigurationSets) != 2 {
		t.Fatalf("expected both shared configuration sets, got %#v", stub.createConfigurationSets)
	}
	if stub.createConfigurationSets[0] != TransactionalConfigurationSet || stub.createConfigurationSets[1] != MarketingConfigurationSet {
		t.Fatalf("unexpected shared configuration sets: %#v", stub.createConfigurationSets)
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
