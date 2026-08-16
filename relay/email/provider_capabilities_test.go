package email_test

import (
	"testing"

	"github.com/dugble/relay/email"
)

func TestProviderSpecificCapabilityConstraints(t *testing.T) {
	base := email.Message{
		From:    email.Address{Email: "sender@example.com"},
		To:      []email.Address{{Email: "one@example.com"}},
		Subject: "Hello",
		Text:    "hello",
	}

	requiresSubject := email.Capabilities{RequiresSubject: true}
	withoutSubject := base
	withoutSubject.Subject = ""
	if requiresSubject.Supports(withoutSubject) {
		t.Fatal("message without subject should not satisfy RequiresSubject")
	}
	if !requiresSubject.Supports(base) {
		t.Fatal("message with subject should satisfy RequiresSubject")
	}

	maxTwo := email.Capabilities{MultipleRecipients: true, MaxRecipients: 2}
	two := base
	two.To = []email.Address{{Email: "one@example.com"}, {Email: "two@example.com"}}
	if !maxTwo.Supports(two) {
		t.Fatal("two recipients should satisfy MaxRecipients=2")
	}
	three := two
	three.To = append(three.To, email.Address{Email: "three@example.com"})
	if maxTwo.Supports(three) {
		t.Fatal("three recipients should exceed MaxRecipients=2")
	}
}
