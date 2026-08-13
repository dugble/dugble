package unsubscribe

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

const path = "/unsubscribe"

// Linker creates and verifies opaque, recipient-specific unsubscribe links.
// The signing key is server-owned and must not be exposed to API callers.
type Linker struct {
	baseURL string
	key     []byte
}

func NewLinker(baseURL string, key []byte) (*Linker, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	parsed, err := url.ParseRequestURI(baseURL)
	if err != nil || parsed.Scheme != "https" || strings.TrimSpace(parsed.Host) == "" {
		return nil, errors.New("unsubscribe base URL must be an absolute HTTPS URL")
	}
	if len(key) < 32 {
		return nil, errors.New("unsubscribe signing key must contain at least 32 bytes")
	}
	return &Linker{baseURL: baseURL, key: append([]byte(nil), key...)}, nil
}

func (linker *Linker) Link(recipientID uuid.UUID) (string, error) {
	if linker == nil || len(linker.key) == 0 || recipientID == uuid.Nil {
		return "", errors.New("unsubscribe linker is not configured")
	}
	payload := recipientID[:]
	mac := hmac.New(sha256.New, linker.key)
	_, _ = mac.Write(payload)
	token := base64.RawURLEncoding.EncodeToString(append(append([]byte(nil), payload...), mac.Sum(nil)...))
	return linker.baseURL + path + "?token=" + token, nil
}

func (linker *Linker) Verify(token string) (uuid.UUID, error) {
	if linker == nil || len(linker.key) == 0 {
		return uuid.Nil, errors.New("unsubscribe linker is not configured")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil || len(decoded) != 16+sha256.Size {
		return uuid.Nil, errors.New("invalid unsubscribe token")
	}
	payload, signature := decoded[:16], decoded[16:]
	mac := hmac.New(sha256.New, linker.key)
	_, _ = mac.Write(payload)
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return uuid.Nil, errors.New("invalid unsubscribe token")
	}
	recipientID, err := uuid.FromBytes(payload)
	if err != nil {
		return uuid.Nil, fmt.Errorf("decode unsubscribe recipient: %w", err)
	}
	return recipientID, nil
}
