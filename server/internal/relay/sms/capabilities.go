package sms

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Capabilities describes provider constraints that can be evaluated before an
// SMS submission is attempted.
type Capabilities struct {
	AlphanumericSenderID  bool
	MaxSenderIDLength     int
	RequiresE164Recipient bool
}

// CapabilityProvider exposes preflight constraints in addition to Provider.
type CapabilityProvider interface {
	Provider
	Capabilities() Capabilities
}

// Supports reports whether a message satisfies the provider's declared
// preflight constraints. It does not make delivery or regulatory guarantees.
func (c Capabilities) Supports(message Message) bool {
	from := strings.TrimSpace(message.From)
	if c.MaxSenderIDLength > 0 && utf8.RuneCountInString(from) > c.MaxSenderIDLength {
		return false
	}
	if from != "" && containsLetter(from) && !c.AlphanumericSenderID {
		return false
	}
	if c.RequiresE164Recipient && !isE164(strings.TrimSpace(message.To)) {
		return false
	}
	return true
}

func containsLetter(value string) bool {
	for _, r := range value {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func isE164(value string) bool {
	if len(value) < 9 || len(value) > 16 || value[0] != '+' || value[1] == '0' {
		return false
	}
	for i := 1; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}
