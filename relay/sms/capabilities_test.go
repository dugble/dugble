package sms_test

import (
	"testing"

	"github.com/dugble/relay/sms"
)

func TestCapabilitiesSupportsPreflightConstraints(t *testing.T) {
	capabilities := sms.Capabilities{
		AlphanumericSenderID:  true,
		MaxSenderIDLength:     11,
		RequiresE164Recipient: true,
	}

	tests := []struct {
		name    string
		message sms.Message
		want    bool
	}{
		{name: "supported", message: sms.Message{To: "+233200000000", From: "Acme", Text: "hello"}, want: true},
		{name: "sender too long", message: sms.Message{To: "+233200000000", From: "TwelveChars12", Text: "hello"}, want: false},
		{name: "recipient not E164", message: sms.Message{To: "0200000000", From: "Acme", Text: "hello"}, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := capabilities.Supports(test.message); got != test.want {
				t.Fatalf("Supports() = %v, want %v", got, test.want)
			}
		})
	}
}
