package domain

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/dugble/dugble/server/internal/modules/emailtenant"
	platformemail "github.com/dugble/dugble/server/internal/platform/awsses"
)

type checkProvider struct {
	status             platformemail.DomainStatus
	associationCalls   int
	associatedDomain   string
	associatedRegion   string
	associatedTenant   string
	associationFailure error
}

func (p *checkProvider) ProvisionDomain(context.Context, platformemail.DomainProvisionRequest) ([]platformemail.VerificationRecord, error) {
	return nil, nil
}

func (p *checkProvider) GetDomainStatus(context.Context, string, string) (platformemail.DomainStatus, error) {
	return p.status, nil
}

func (p *checkProvider) DeleteDomain(context.Context, string, string) error { return nil }

func (p *checkProvider) AssociateDomainWithTenant(_ context.Context, domainName, region, tenantName string) error {
	p.associationCalls++
	p.associatedDomain = domainName
	p.associatedRegion = region
	p.associatedTenant = tenantName
	return p.associationFailure
}

type verifiedDNS struct{}

func (verifiedDNS) Verify(context.Context, string, platformemail.VerificationRecord) bool {
	return true
}

func TestCheckAssociatesVerifiedDomainWithTenant(t *testing.T) {
	teamID := uuid.New()
	provider := &checkProvider{status: platformemail.DomainStatus{
		IdentityVerified: true,
		DKIMVerified:     true,
		MailFromVerified: true,
	}}
	service := NewService(nil, provider, verifiedDNS{})
	domain := SenderDomain{
		TeamID:         teamID.String(),
		Domain:         "mail.example.com",
		ProviderRegion: "us-east-1",
		VerificationRecords: []VerificationRecord{
			{Record: platformemail.RecordDKIM, Name: "selector._domainkey", Type: platformemail.RecordTypeTXT},
			{Record: platformemail.RecordSPF, Name: "send", Type: platformemail.RecordTypeMX},
		},
	}

	result, err := service.Check(context.Background(), domain)
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if result.Status != StatusVerified {
		t.Fatalf("expected status %q, got %q", StatusVerified, result.Status)
	}
	if provider.associationCalls != 1 {
		t.Fatalf("expected one tenant association, got %d", provider.associationCalls)
	}
	if provider.associatedDomain != domain.Domain {
		t.Fatalf("expected domain %q, got %q", domain.Domain, provider.associatedDomain)
	}
	if provider.associatedRegion != domain.ProviderRegion {
		t.Fatalf("expected region %q, got %q", domain.ProviderRegion, provider.associatedRegion)
	}
	expectedTenant := emailtenant.AWSExternalName(teamID)
	if provider.associatedTenant != expectedTenant {
		t.Fatalf("expected tenant %q, got %q", expectedTenant, provider.associatedTenant)
	}
}

func TestCheckDoesNotAssociatePendingDomain(t *testing.T) {
	provider := &checkProvider{status: platformemail.DomainStatus{
		IdentityVerified: false,
		DKIMVerified:     false,
		MailFromVerified: false,
	}}
	service := NewService(nil, provider, verifiedDNS{})
	domain := SenderDomain{
		TeamID:         uuid.NewString(),
		Domain:         "mail.example.com",
		ProviderRegion: "us-east-1",
		VerificationRecords: []VerificationRecord{
			{Record: platformemail.RecordDKIM, Name: "selector._domainkey", Type: platformemail.RecordTypeTXT},
		},
	}

	result, err := service.Check(context.Background(), domain)
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if result.Status != StatusPending {
		t.Fatalf("expected status %q, got %q", StatusPending, result.Status)
	}
	if provider.associationCalls != 0 {
		t.Fatalf("expected no tenant association, got %d", provider.associationCalls)
	}
}
