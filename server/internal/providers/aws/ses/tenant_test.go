package ses

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sestypes "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/aws/smithy-go"

	"github.com/dugble/dugble/server/internal/modules/emailtenant"
)

type tenantClientStub struct {
	createCalls      int
	getCalls         int
	suppressionCalls int
	associationCalls int
	policyCalls      int
}

func (stub *tenantClientStub) CreateTenant(context.Context, *sesv2.CreateTenantInput, ...func(*sesv2.Options)) (*sesv2.CreateTenantOutput, error) {
	stub.createCalls++
	return nil, &smithy.GenericAPIError{Code: "AlreadyExistsException", Message: "tenant already exists", Fault: smithy.FaultClient}
}

func (stub *tenantClientStub) GetTenant(context.Context, *sesv2.GetTenantInput, ...func(*sesv2.Options)) (*sesv2.GetTenantOutput, error) {
	stub.getCalls++
	return &sesv2.GetTenantOutput{Tenant: &sestypes.Tenant{
		TenantId:   aws.String("tenant-id"),
		TenantArn:  aws.String("arn:aws:ses:eu-north-1:123456789012:tenant/customer"),
		TenantName: aws.String("customer"),
	}}, nil
}

func (stub *tenantClientStub) PutTenantSuppressionAttributes(context.Context, *sesv2.PutTenantSuppressionAttributesInput, ...func(*sesv2.Options)) (*sesv2.PutTenantSuppressionAttributesOutput, error) {
	stub.suppressionCalls++
	return &sesv2.PutTenantSuppressionAttributesOutput{}, nil
}

func (stub *tenantClientStub) CreateTenantResourceAssociation(context.Context, *sesv2.CreateTenantResourceAssociationInput, ...func(*sesv2.Options)) (*sesv2.CreateTenantResourceAssociationOutput, error) {
	stub.associationCalls++
	if stub.associationCalls == 1 {
		return nil, &smithy.GenericAPIError{Code: "AlreadyExistsException", Message: "association already exists", Fault: smithy.FaultClient}
	}
	return &sesv2.CreateTenantResourceAssociationOutput{}, nil
}

func (stub *tenantClientStub) DeleteTenantResourceAssociation(context.Context, *sesv2.DeleteTenantResourceAssociationInput, ...func(*sesv2.Options)) (*sesv2.DeleteTenantResourceAssociationOutput, error) {
	return &sesv2.DeleteTenantResourceAssociationOutput{}, nil
}

func (stub *tenantClientStub) UpdateReputationEntityPolicy(context.Context, *sesv2.UpdateReputationEntityPolicyInput, ...func(*sesv2.Options)) (*sesv2.UpdateReputationEntityPolicyOutput, error) {
	stub.policyCalls++
	return &sesv2.UpdateReputationEntityPolicyOutput{}, nil
}

func TestProvisionTenantConvergesExistingTenant(t *testing.T) {
	stub := &tenantClientStub{}
	client := &Client{
		defaultRegion: "eu-north-1",
		tenantClients: map[string]sesTenantAPI{"eu-north-1": stub},
	}

	result, err := client.ProvisionTenant(context.Background(), emailtenant.ProvisionRequest{
		Region:           "eu-north-1",
		ExternalName:     "customer",
		SuppressionScope: emailtenant.SuppressionScopeTenant,
		ReputationPolicy: emailtenant.ReputationPolicyStandard,
	})
	if err != nil {
		t.Fatalf("provision existing SES tenant: %v", err)
	}
	if result.ExternalID != "tenant-id" || result.TenantARN != "arn:aws:ses:eu-north-1:123456789012:tenant/customer" {
		t.Fatalf("unexpected provision result: %#v", result)
	}
	if stub.createCalls != 1 || stub.getCalls != 1 {
		t.Fatalf("expected create-or-get convergence, create=%d get=%d", stub.createCalls, stub.getCalls)
	}
	if stub.suppressionCalls != 1 {
		t.Fatalf("expected suppression settings to be converged once, got %d", stub.suppressionCalls)
	}
	if stub.associationCalls != 2 {
		t.Fatalf("expected both shared configuration sets to be converged, got %d associations", stub.associationCalls)
	}
	if stub.policyCalls != 1 {
		t.Fatalf("expected reputation policy to be converged once, got %d", stub.policyCalls)
	}
}
