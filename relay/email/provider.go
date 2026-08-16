package email

import (
	"context"

	relaycore "github.com/dugble/relay"
)

// SubmissionState is an alias of Relay's channel-neutral submission state.
type SubmissionState = relaycore.SubmissionState

const (
	SubmissionAccepted = relaycore.SubmissionAccepted
	SubmissionRejected = relaycore.SubmissionRejected
	SubmissionUnknown  = relaycore.SubmissionUnknown
)

// SendResult is an alias of Relay's channel-neutral provider result.
type SendResult = relaycore.Result

// Provider sends email through one communications provider.
type Provider interface {
	relaycore.Provider
	Send(context.Context, Message) (SendResult, error)
}
