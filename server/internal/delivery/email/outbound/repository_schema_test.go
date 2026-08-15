package emaildelivery

import (
	"os"
	"strings"
	"testing"
)

func TestRepositoryDoesNotReferenceRemovedDomainSendingEnabledColumn(t *testing.T) {
	source, err := os.ReadFile("repository.go")
	if err != nil {
		t.Fatalf("read repository.go: %v", err)
	}
	if strings.Contains(string(source), "sending_enabled") {
		t.Fatal("repository.go references removed domains.sending_enabled column")
	}
}
