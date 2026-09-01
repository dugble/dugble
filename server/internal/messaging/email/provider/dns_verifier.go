package awsses

import "context"

// DNSVerifier checks whether a provider-neutral verification record is present.
// Concrete DNS resolution belongs in internal/integrations/dns.
type DNSVerifier interface {
	Verify(context.Context, string, VerificationRecord) bool
}
