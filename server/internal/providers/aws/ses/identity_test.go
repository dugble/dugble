package ses

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/smithy-go"
)

type identityClientStub struct {
	createCalls int
	dkimCalls   int
	dkimInput   *sesv2.PutEmailIdentityDkimSigningAttributesInput
}

func (stub *identityClientStub) CreateEmailIdentity(context.Context, *sesv2.CreateEmailIdentityInput, ...func(*sesv2.Options)) (*sesv2.CreateEmailIdentityOutput, error) {
	stub.createCalls++
	return nil, &smithy.GenericAPIError{Code: "AlreadyExistsException", Message: "identity already exists", Fault: smithy.FaultClient}
}

func (stub *identityClientStub) PutEmailIdentityDkimSigningAttributes(_ context.Context, input *sesv2.PutEmailIdentityDkimSigningAttributesInput, _ ...func(*sesv2.Options)) (*sesv2.PutEmailIdentityDkimSigningAttributesOutput, error) {
	stub.dkimCalls++
	stub.dkimInput = input
	return &sesv2.PutEmailIdentityDkimSigningAttributesOutput{}, nil
}

func (stub *identityClientStub) PutEmailIdentityMailFromAttributes(context.Context, *sesv2.PutEmailIdentityMailFromAttributesInput, ...func(*sesv2.Options)) (*sesv2.PutEmailIdentityMailFromAttributesOutput, error) {
	return &sesv2.PutEmailIdentityMailFromAttributesOutput{}, nil
}

func (stub *identityClientStub) GetEmailIdentity(context.Context, *sesv2.GetEmailIdentityInput, ...func(*sesv2.Options)) (*sesv2.GetEmailIdentityOutput, error) {
	return &sesv2.GetEmailIdentityOutput{}, nil
}

func (stub *identityClientStub) DeleteEmailIdentity(context.Context, *sesv2.DeleteEmailIdentityInput, ...func(*sesv2.Options)) (*sesv2.DeleteEmailIdentityOutput, error) {
	return &sesv2.DeleteEmailIdentityOutput{}, nil
}

func TestProvisionDomainRefreshesBYODKIMForExistingIdentity(t *testing.T) {
	stub := &identityClientStub{}
	client := &Client{
		defaultRegion: "eu-north-1",
		identityClients: map[string]sesIdentityAPI{
			"eu-north-1": stub,
		},
	}

	records, err := client.ProvisionDomain(context.Background(), DomainProvisionRequest{
		Domain:           "example.com",
		Region:           "eu-north-1",
		CustomReturnPath: "send",
		SESTenantName:    "customer",
	})
	if err != nil {
		t.Fatalf("provision existing SES identity: %v", err)
	}
	if stub.createCalls != 1 || stub.dkimCalls != 1 {
		t.Fatalf("expected create then BYODKIM refresh, create=%d dkim=%d", stub.createCalls, stub.dkimCalls)
	}
	if stub.dkimInput == nil || strings.TrimSpace(aws.ToString(stub.dkimInput.EmailIdentity)) != "example.com" {
		t.Fatalf("unexpected DKIM refresh input: %#v", stub.dkimInput)
	}
	if stub.dkimInput.SigningAttributes == nil || strings.TrimSpace(aws.ToString(stub.dkimInput.SigningAttributes.DomainSigningSelector)) == "" {
		t.Fatal("expected fresh BYODKIM signing attributes")
	}
	if len(records) != 3 || records[0].Record != RecordDKIM || records[0].Status != RecordStatusPending {
		t.Fatalf("unexpected verification records: %#v", records)
	}
}
