package moolre

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/dugble/dugble/server/internal/relay/sms"
)

var ErrInvalidConfig = errors.New("invalid Moolre configuration")

// Config configures the Moolre provider.
type Config struct {
	VASKey     string
	BaseURL    string
	HTTPClient *http.Client
}

// Provider owns communication with Moolre.
type Provider struct {
	client *client
}

func New(config Config) (*Provider, error) {
	client, err := newClient(config)
	if err != nil {
		return nil, err
	}
	return &Provider{client: client}, nil
}

func (p *Provider) Name() string { return "moolre" }

func (p *Provider) Capabilities() sms.Capabilities {
	return sms.Capabilities{
		AlphanumericSenderID: true,
		MaxSenderIDLength:    11,
	}
}

// APIError records a non-success response returned by Moolre.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	message := strings.TrimSpace(e.Message)
	if message == "" {
		message = http.StatusText(e.StatusCode)
	}
	if e.Code != "" {
		return fmt.Sprintf("Moolre returned HTTP %d code %s: %s", e.StatusCode, e.Code, message)
	}
	return fmt.Sprintf("Moolre returned HTTP %d: %s", e.StatusCode, message)
}
