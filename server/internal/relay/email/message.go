package email

import (
	"net/mail"
	"strings"

	relaycore "github.com/dugble/dugble/server/internal/relay"
)

// Address is a mailbox used by an email message.
type Address struct {
	Email string
	Name  string
}

// Message is the provider-neutral transactional email request accepted by Relay.
type Message struct {
	From    Address
	To      []Address
	ReplyTo *Address
	Subject string
	Text    string
	HTML    string
}

func (m Message) validate() error {
	if !validAddress(m.From) || len(m.To) == 0 {
		return relaycore.ErrInvalidMessage
	}
	for _, recipient := range m.To {
		if !validAddress(recipient) {
			return relaycore.ErrInvalidMessage
		}
	}
	if m.ReplyTo != nil && !validAddress(*m.ReplyTo) {
		return relaycore.ErrInvalidMessage
	}
	if strings.TrimSpace(m.Text) == "" && strings.TrimSpace(m.HTML) == "" {
		return relaycore.ErrInvalidMessage
	}
	return nil
}

func validAddress(address Address) bool {
	value := strings.TrimSpace(address.Email)
	if value == "" {
		return false
	}
	parsed, err := mail.ParseAddress(value)
	return err == nil && parsed.Address == value
}
