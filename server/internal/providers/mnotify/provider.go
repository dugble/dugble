package mnotify

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/dugble/dugble/server/internal/relay/sms"
)

const defaultBaseURL = "https://api.mnotify.com/api/sms/quick"

var ErrInvalidConfig = errors.New("invalid mNotify configuration")

// Config configures the mNotify provider.
type Config struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

// Provider owns communication with mNotify.
type Provider struct {
	client *client
}

func New(config Config) (*Provider, error) {
	apiKey := strings.TrimSpace(config.APIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("%w: API key is required", ErrInvalidConfig)
	}

	base := strings.TrimSpace(config.BaseURL)
	if base == "" {
		base = defaultBaseURL
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("%w: invalid base URL", ErrInvalidConfig)
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Provider{client: &client{apiKey: apiKey, baseURL: parsed, http: httpClient}}, nil
}

func (p *Provider) Name() string { return "mnotify" }

func (p *Provider) Capabilities() sms.Capabilities {
	return sms.Capabilities{
		AlphanumericSenderID: true,
		MaxSenderIDLength:    11,
	}
}

// APIError records a non-success response returned by mNotify.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("mNotify returned HTTP %d code %s: %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("mNotify returned HTTP %d: %s", e.StatusCode, e.Message)
}
