package ses

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/smithy-go"

	platformemail "github.com/dugble/dugble/server/internal/platform/awsses"
)

type domainIdentityStub struct {
	createErr   error
	dkimErr     error
	dkimInput   *sesv2.PutEmailIdentityDkimSigningAttributesInput
	createCalls int
	dkimCalls   int
}

func (stub *domainIdentityStub) CreateEmailIdentity(
	context.Context,
	*sesv2.CreateEmailIdentityInput,
	...func(*sesv2.Options),
) (*sesv2.CreateEmailIdentityOutput, error) {
	stub.createCalls++
	if stub.createErr != nil {
		return nil, stub.createErr
	}
	return &sesv2.CreateEmailIdentityOutput{}, nil
}

func (stub *domainIdentityStub) PutEmailIdentityDkimSigningAttributes(
	_ context.Context,
	input *sesv2.PutEmailIdentityDkimSigningAttributesInput,
	_ ...func(*sesv2.Options),
) (*sesv2.PutEmailIdentityDkimSigningAttributesOutput, error) {
	stub.dkimCalls++
	stub.dkimInput = input
	if stub.dkimErr != nil {
		return nil, stub.dkimErr
	}
	return &sesv2.PutEmailIdentityDkimSigningAttributesOutput{}, nil
}

func (stub *domainIdentityStub) PutEmailIdentityMailFromAttributes(
	context.Context,
	*sesv2.PutEmailIdentityMailFromAttributesInput,
	...func(*sesv2.Options),
) (*sesv2.PutEmailIdentityMailFromAttributesOutput, error) {
	return &sesv2.PutEmailIdentityMailFromAttributesOutput{}, nil
}

func (stub *domainIdentityStub) GetEmailIdentity(
	context.Context,
	*sesv2.GetEmailIdentityInput,
	...func(*sesv2.Options),
) (*sesv2.GetEmailIdentityOutput, error) {
	return &sesv2.GetEmailIdentityOutput{}, nil
}

func (stub *domainIdentityStub) DeleteEmailIdentity(
	context.Context,
	*sesv2.DeleteEmailIdentityInput,
	...func(*sesv2.Options),
) (*sesv2.DeleteEmailIdentityOutput, error) {
	return &sesv2.DeleteEmailIdentityOutput{}, nil
}

func TestProvisionDomainRecoversExistingIdentityWithFreshBYODKIM(t *testing.T) {
	stub := &domainIdentityStub{createErr: &smithy.GenericAPIError{
		Code:    "AlreadyExistsException",
		Message: "identity already exists",
		Fault:   smithy.FaultClient,
	}}
	client := &Client{
		defaultRegion: "eu-west-1",
		identityClients: map[string]sesIdentityAPI{
			"eu-west-1": stub,
		},
	}

	records, err := client.ProvisionDomain(context.Background(), platformemail.DomainProvisionRequest{
		Domain:           "example.com",
		Region:           "eu-west-1",
		CustomReturnPath: "send",
		SESTenantName:    "tenant-123",
	})
	if err != nil {
		t.Fatalf("provision existing SES domain: %v", err)
	}
	if stub.createCalls != 1 {
		t.Fatalf("expected one identity create attempt, got %d", stub.createCalls)
	}
	if stub.dkimCalls != 1 {
		t.Fatalf("expected one DKIM recovery call, got %d", stub.dkimCalls)
	}
	if stub.dkimInput == nil {
		t.Fatal("expected DKIM recovery input")
	}
	if got := strings.TrimSpace(aws.ToString(stub.dkimInput.EmailIdentity)); got != "example.com" {
		t.Fatalf("expected DKIM identity example.com, got %q", got)
	}
	if stub.dkimInput.SigningAttributes == nil {
		t.Fatal("expected external DKIM signing attributes")
	}
	selector := strings.TrimSpace(aws.ToString(stub.dkimInput.SigningAttributes.DomainSigningSelector))
	if selector == "" {
		t.Fatal("expected generated DKIM selector")
	}
	if len(records) == 0 || records[0].Record != platformemail.RecordDKIM {
		t.Fatalf("expected DKIM verification record, got %#v", records)
	}
	if records[0].Name != selector+"._domainkey" {
		t.Fatalf("expected verification selector %q, got %q", selector, records[0].Name)
	}
}

func TestProvisionDomainReturnsDKIMRecoveryError(t *testing.T) {
	expected := errors.New("cannot update DKIM")
	stub := &domainIdentityStub{
		createErr: &smithy.GenericAPIError{
			Code:    "AlreadyExistsException",
			Message: "identity already exists",
			Fault:   smithy.FaultClient,
		},
		dkimErr: expected,
	}
	client := &Client{
		defaultRegion: "eu-west-1",
		identityClients: map[string]sesIdentityAPI{
			"eu-west-1": stub,
		},
	}

	_, err := client.ProvisionDomain(context.Background(), platformemail.DomainProvisionRequest{
		Domain:           "example.com",
		Region:           "eu-west-1",
		CustomReturnPath: "send",
		SESTenantName:    "tenant-123",
	})
	if err == nil {
		t.Fatal("expected DKIM recovery error")
	}
	if !errors.Is(err, expected) {
		t.Fatalf("expected wrapped DKIM error, got %v", err)
	}
}
