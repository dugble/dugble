package ses

import "context"

const (
	RecordDKIM = "DKIM"
	RecordSPF  = "SPF"

	RecordTypeTXT = "TXT"
	RecordTypeMX  = "MX"

	RecordStatusPending  = "pending"
	RecordStatusVerified = "verified"
	RecordStatusFailed   = "failed"
)

type VerificationRecord struct {
	Record   string `json:"record"`
	Name     string `json:"name"`
	Value    string `json:"value"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	TTL      string `json:"ttl"`
	Priority *int   `json:"priority,omitempty"`
}

// DNSVerifier checks whether a verification record is present. Concrete DNS
// resolution remains outside the SES provider package.
type DNSVerifier interface {
	Verify(context.Context, string, VerificationRecord) bool
}
