package senderidreconciliation

import (
	"testing"

	platformsenderid "github.com/dugble/dugble/server/internal/platform/senderid"
)

func TestNotifiableStatus(t *testing.T) {
	for _, status := range []string{platformsenderid.StatusApproved, platformsenderid.StatusRejected, platformsenderid.StatusSuspended} {
		if !notifiableStatus(status) {
			t.Fatalf("notifiableStatus(%q) = false", status)
		}
	}
	for _, status := range []string{platformsenderid.StatusPending, platformsenderid.StatusUnknown, ""} {
		if notifiableStatus(status) {
			t.Fatalf("notifiableStatus(%q) = true", status)
		}
	}
}
