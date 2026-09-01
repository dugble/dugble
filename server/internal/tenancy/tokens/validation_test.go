package teamtoken

import (
	"testing"
	"time"
)

func TestNormalizeExpiresAtAllowsNonExpiringToken(t *testing.T) {
	expiresAt, err := normalizeExpiresAt(nil)
	if err != nil {
		t.Fatalf("normalizeExpiresAt(nil) returned error: %v", err)
	}
	if expiresAt != nil {
		t.Fatalf("normalizeExpiresAt(nil) returned %v, want nil", expiresAt)
	}
}

func TestNormalizeExpiresAtRejectsExpiredToken(t *testing.T) {
	expiresAt := time.Now().UTC().Add(-time.Minute)
	if _, err := normalizeExpiresAt(&expiresAt); err == nil {
		t.Fatal("normalizeExpiresAt(expired) returned nil error")
	}
}

func TestNormalizeExpiresAtRejectsTokenBeyondMaximumTTL(t *testing.T) {
	expiresAt := time.Now().UTC().Add(maxTokenTTL + time.Minute)
	if _, err := normalizeExpiresAt(&expiresAt); err == nil {
		t.Fatal("normalizeExpiresAt(beyond max TTL) returned nil error")
	}
}

func TestNormalizeExpiresAtPreservesValidExpiration(t *testing.T) {
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)
	got, err := normalizeExpiresAt(&expiresAt)
	if err != nil {
		t.Fatalf("normalizeExpiresAt(valid) returned error: %v", err)
	}
	if got == nil {
		t.Fatal("normalizeExpiresAt(valid) returned nil")
	}
	if !got.Equal(expiresAt) {
		t.Fatalf("normalizeExpiresAt(valid) returned %v, want %v", got, expiresAt)
	}
}
