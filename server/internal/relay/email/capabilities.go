package email

import "strings"

// Capabilities describes provider constraints that can be evaluated before an
// email submission is attempted.
type Capabilities struct {
	HTML               bool
	ReplyTo            bool
	MultipleRecipients bool
	RequiresSubject    bool
	MaxRecipients      int
}

// CapabilityProvider exposes preflight constraints in addition to Provider.
type CapabilityProvider interface {
	Provider
	Capabilities() Capabilities
}

// Supports reports whether a message satisfies the provider's declared
// preflight constraints. It does not make delivery or policy guarantees.
func (c Capabilities) Supports(message Message) bool {
	if strings.TrimSpace(message.HTML) != "" && !c.HTML {
		return false
	}
	if message.ReplyTo != nil && !c.ReplyTo {
		return false
	}
	if len(message.To) > 1 && !c.MultipleRecipients {
		return false
	}
	if c.RequiresSubject && strings.TrimSpace(message.Subject) == "" {
		return false
	}
	if c.MaxRecipients > 0 && len(message.To) > c.MaxRecipients {
		return false
	}
	return true
}
