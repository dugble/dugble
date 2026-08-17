package sms

import (
	"strings"

	relaycore "github.com/dugble/dugble/server/internal/relay"
)

// ErrInvalidMessage is retained for source compatibility. New code may use
// relay.ErrInvalidMessage directly.
var ErrInvalidMessage = relaycore.ErrInvalidMessage

// Purpose describes why a message is being sent. Routing policies may use it
// to choose providers differently for verification and transactional traffic.
type Purpose string

const (
	PurposeTransactional Purpose = "transactional"
	PurposeVerification  Purpose = "verification"
)

// Message is the provider-neutral SMS request accepted by Relay.
type Message struct {
	Reference string
	To        string
	From      string
	Text      string
	Purpose   Purpose
}

func (m Message) validate() error {
	if strings.TrimSpace(m.To) == "" {
		return ErrInvalidMessage
	}
	if strings.TrimSpace(m.Text) == "" {
		return ErrInvalidMessage
	}
	return nil
}
