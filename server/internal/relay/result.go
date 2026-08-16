package relay

// SubmissionState captures what Relay knows about provider acceptance.
type SubmissionState string

const (
	SubmissionAccepted SubmissionState = "accepted"
	SubmissionRejected SubmissionState = "rejected"
	SubmissionUnknown  SubmissionState = "unknown"
)

// Normalize returns a supported submission state. Empty or unrecognized values
// are treated as unknown so callers never infer a safe fallback from ambiguity.
func (s SubmissionState) Normalize() SubmissionState {
	switch s {
	case SubmissionAccepted, SubmissionRejected, SubmissionUnknown:
		return s
	default:
		return SubmissionUnknown
	}
}

// Result is the channel-neutral normalized result of one provider submission.
type Result struct {
	Provider          string
	ProviderMessageID string
	ProviderStatus    string
	State             SubmissionState
}
