package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const DefaultTokenBytes = 32

var ErrInvalidTokenSize = errors.New("token size must be greater than zero")

// NewToken returns a cryptographically secure URL-safe opaque token.
func NewToken() (string, error) { return NewTokenWithSize(DefaultTokenBytes) }

func NewTokenWithSize(size int) (string, error) {
	if size <= 0 {
		return "", ErrInvalidTokenSize
	}
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

// HashToken returns a stable, non-reversible representation suitable for
// storing opaque tokens in a database.
func HashToken(token string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

// NewSessionToken preserves the existing fixed-size session token format.
func NewSessionToken() (string, error) {
	var token [DefaultTokenBytes]byte
	if _, err := rand.Read(token[:]); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(token[:]), nil
}

// HashSessionToken preserves the existing hexadecimal session-token digest.
func HashSessionToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}
