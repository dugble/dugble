package unsubscribe

import (
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestLinkRoundTripAndTamperRejection(t *testing.T) {
	t.Parallel()

	linker, err := NewLinker("https://api.dugble.com/", []byte(strings.Repeat("k", 32)))
	if err != nil {
		t.Fatalf("NewLinker() error = %v", err)
	}
	recipientID := uuid.New()
	link, err := linker.Link(recipientID)
	if err != nil {
		t.Fatalf("Link() error = %v", err)
	}
	parsed, err := url.Parse(link)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	token := parsed.Query().Get("token")
	verified, err := linker.Verify(token)
	if err != nil || verified != recipientID {
		t.Fatalf("Verify() = %s, %v; want %s", verified, err, recipientID)
	}
	replacement := "A"
	if strings.HasSuffix(token, replacement) {
		replacement = "B"
	}
	if _, err := linker.Verify(token[:len(token)-1] + replacement); err == nil {
		t.Fatal("Verify() accepted a modified token")
	}
}
