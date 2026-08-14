package ses

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sesv2types "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/aws/smithy-go"

	platformemail "github.com/dugble/dugble/server/internal/platform/awsses"
)

func (c *Client) ProvisionDomain(ctx context.Context, req platformemail.DomainProvisionRequest) ([]platformemail.VerificationRecord, error) {
	client, err := c.identityClient(req.Region)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.SESTenantName) == "" {
		return nil, errors.New("SES tenant name is required for sender-domain provisioning")
	}
	selector, privateKey, publicKey, err := generateBYODKIMMaterial()
	if err != nil {
		return nil, fmt.Errorf("generate BYODKIM material: %w", err)
	}
	_, err = client.CreateEmailIdentity(ctx, &sesv2.CreateEmailIdentityInput{
		EmailIdentity: aws.String(req.Domain),
		DkimSigningAttributes: &sesv2types.DkimSigningAttributes{
			DomainSigningPrivateKey: aws.String(privateKey),
			DomainSigningSelector:   aws.String(selector),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create SES email identity: %w", err)
	}
	return mapVerificationRecords(req, selector, publicKey), nil
}

// ConfigureDomainMailFrom configures the custom MAIL FROM domain after SES has
// verified the sender identity. SES documents this operation against a verified
// identity, so it must not run synchronously with CreateEmailIdentity.
func (c *Client) ConfigureDomainMailFrom(ctx context.Context, domainName, region, customReturnPath string) error {
	client, err := c.identityClient(region)
	if err != nil {
		return err
	}
	_, err = client.PutEmailIdentityMailFromAttributes(ctx, &sesv2.PutEmailIdentityMailFromAttributesInput{
		EmailIdentity:       aws.String(strings.TrimSpace(domainName)),
		MailFromDomain:      aws.String(strings.TrimSpace(customReturnPath) + "." + strings.TrimSpace(domainName)),
		BehaviorOnMxFailure: sesv2types.BehaviorOnMxFailureRejectMessage,
	})
	if err != nil {
		return fmt.Errorf("configure SES MAIL FROM domain: %w", err)
	}
	return nil
}

// AssociateDomainWithTenant associates a verified sender identity with the
// customer's SES tenant. It is intentionally separate from ProvisionDomain:
// SES tenant identities are expected to be verified resources, while a newly
// created identity is still pending DNS verification.
func (c *Client) AssociateDomainWithTenant(ctx context.Context, domainName, region, tenantName string) error {
	client, err := c.tenantClient(region)
	if err != nil {
		return err
	}
	tenantName = strings.TrimSpace(tenantName)
	if tenantName == "" {
		return errors.New("SES tenant name is required for sender identity association")
	}
	resourceARN, err := c.tenantIdentityARN(ctx, client, tenantName, domainName)
	if err != nil {
		return err
	}
	_, err = client.CreateTenantResourceAssociation(ctx, &sesv2.CreateTenantResourceAssociationInput{
		TenantName:  aws.String(tenantName),
		ResourceArn: aws.String(resourceARN),
	})
	if err != nil && !isAlreadyExists(err) {
		return fmt.Errorf("associate SES sender identity with tenant: %w", err)
	}
	return nil
}

// DisassociateDomainFromTenant removes a tenant-resource association before the
// underlying SES identity is deleted. SES rejects deletion of associated
// identities, so this cleanup is part of the provider deletion lifecycle.
func (c *Client) DisassociateDomainFromTenant(ctx context.Context, domainName, region, tenantName string) error {
	client, err := c.tenantClient(region)
	if err != nil {
		return err
	}
	tenantName = strings.TrimSpace(tenantName)
	if tenantName == "" {
		return errors.New("SES tenant name is required for sender identity disassociation")
	}
	resourceARN, err := c.tenantIdentityARN(ctx, client, tenantName, domainName)
	if err != nil {
		if isSESNotFound(err) {
			return nil
		}
		return err
	}
	_, err = client.DeleteTenantResourceAssociation(ctx, &sesv2.DeleteTenantResourceAssociationInput{
		TenantName:  aws.String(tenantName),
		ResourceArn: aws.String(resourceARN),
	})
	if err != nil && !isSESNotFound(err) {
		return fmt.Errorf("disassociate SES sender identity from tenant: %w", err)
	}
	return nil
}

func (c *Client) tenantIdentityARN(ctx context.Context, client sesTenantAPI, tenantName, domainName string) (string, error) {
	output, err := client.GetTenant(ctx, &sesv2.GetTenantInput{TenantName: aws.String(strings.TrimSpace(tenantName))})
	if err != nil {
		return "", fmt.Errorf("get SES tenant for sender identity: %w", err)
	}
	if output.Tenant == nil || output.Tenant.TenantArn == nil {
		return "", errors.New("SES returned an incomplete tenant for sender identity association")
	}
	return identityARN(strings.TrimSpace(*output.Tenant.TenantArn), domainName)
}

func identityARN(tenantARN, identity string) (string, error) {
	parts := strings.Split(tenantARN, ":")
	identity = strings.ToLower(strings.TrimSpace(identity))
	if len(parts) < 6 || parts[0] != "arn" || parts[2] != "ses" || strings.TrimSpace(parts[3]) == "" || strings.TrimSpace(parts[4]) == "" {
		return "", fmt.Errorf("invalid SES tenant ARN %q", tenantARN)
	}
	if identity == "" {
		return "", errors.New("SES identity is required")
	}
	return fmt.Sprintf("arn:%s:ses:%s:%s:identity/%s", parts[1], parts[3], parts[4], identity), nil
}

func (c *Client) GetDomainStatus(ctx context.Context, domainName, region string) (platformemail.DomainStatus, error) {
	client, err := c.identityClient(region)
	if err != nil {
		return platformemail.DomainStatus{}, err
	}
	output, err := client.GetEmailIdentity(ctx, &sesv2.GetEmailIdentityInput{EmailIdentity: aws.String(domainName)})
	if err != nil {
		return platformemail.DomainStatus{}, fmt.Errorf("get SES email identity: %w", err)
	}
	status := platformemail.DomainStatus{IdentityVerified: output.VerifiedForSendingStatus}
	if output.DkimAttributes != nil {
		status.DKIMVerified = output.DkimAttributes.SigningEnabled && output.DkimAttributes.Status == sesv2types.DkimStatusSuccess
	}
	if output.MailFromAttributes != nil {
		status.MailFromConfigured = strings.TrimSpace(aws.ToString(output.MailFromAttributes.MailFromDomain)) != ""
		status.MailFromVerified = output.MailFromAttributes.MailFromDomainStatus == sesv2types.MailFromDomainStatusSuccess
	}
	return status, nil
}

func (c *Client) DeleteDomain(ctx context.Context, domainName, region string) error {
	client, err := c.identityClient(region)
	if err != nil {
		return err
	}
	if err := c.deleteDomainWithClient(ctx, client, domainName); err != nil {
		return fmt.Errorf("delete SES email identity: %w", err)
	}
	return nil
}

func (c *Client) deleteDomainWithClient(ctx context.Context, client sesIdentityAPI, domainName string) error {
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		_, err := client.DeleteEmailIdentity(ctx, &sesv2.DeleteEmailIdentityInput{EmailIdentity: aws.String(strings.TrimSpace(domainName))})
		if err == nil || isSESNotFound(err) {
			return nil
		}
		lastErr = err
		if !isSESTransientMutation(err) || attempt == 3 {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 150 * time.Millisecond):
		}
	}
	return lastErr
}

func isSESNotFound(err error) bool {
	var apiError smithy.APIError
	return errors.As(err, &apiError) && strings.EqualFold(strings.TrimSpace(apiError.ErrorCode()), "NotFoundException")
}

func isSESTransientMutation(err error) bool {
	var apiError smithy.APIError
	if !errors.As(err, &apiError) {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(apiError.ErrorCode())) {
	case "concurrentmodificationexception", "toomanyrequestsexception":
		return true
	default:
		return false
	}
}

func mapVerificationRecords(req platformemail.DomainProvisionRequest, selector, publicKey string) []platformemail.VerificationRecord {
	priority := 10
	return []platformemail.VerificationRecord{
		{Record: platformemail.RecordDKIM, Name: selector + "._domainkey", Value: "p=" + publicKey, Type: platformemail.RecordTypeTXT, Status: platformemail.RecordStatusPending, TTL: "Auto"},
		{Record: platformemail.RecordSPF, Name: req.CustomReturnPath, Value: "feedback-smtp." + req.Region + ".amazonses.com", Type: platformemail.RecordTypeMX, Status: platformemail.RecordStatusPending, TTL: "Auto", Priority: &priority},
		{Record: platformemail.RecordSPF, Name: req.CustomReturnPath, Value: "v=spf1 include:amazonses.com ~all", Type: platformemail.RecordTypeTXT, Status: platformemail.RecordStatusPending, TTL: "Auto"},
	}
}

func generateBYODKIMMaterial() (selector, privateKey, publicKey string, err error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", "", err
	}
	privateDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", "", "", err
	}
	publicDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return "", "", "", err
	}
	random := make([]byte, 6)
	if _, err := rand.Read(random); err != nil {
		return "", "", "", err
	}
	selector = "dugble" + hex.EncodeToString(random)
	return selector, base64.StdEncoding.EncodeToString(privateDER), base64.StdEncoding.EncodeToString(publicDER), nil
}

func isAlreadyExists(err error) bool {
	var apiError smithy.APIError
	return errors.As(err, &apiError) && strings.EqualFold(apiError.ErrorCode(), "AlreadyExistsException")
}
