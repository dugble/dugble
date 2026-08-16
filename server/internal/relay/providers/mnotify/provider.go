package mnotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/dugble/dugble/server/internal/relay/sms"
)

const defaultBaseURL = "https://api.mnotify.com/api/sms/quick"

var ErrInvalidConfig = errors.New("invalid mNotify configuration")

// Config configures the mNotify SMS adapter.
type Config struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

// Provider sends SMS through mNotify.
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

func (p *Provider) Send(ctx context.Context, message sms.Message) (sms.SendResult, error) {
	if p == nil || p.client == nil {
		return sms.SendResult{State: sms.SubmissionRejected}, ErrInvalidConfig
	}
	if strings.TrimSpace(message.From) == "" || !p.Capabilities().Supports(message) {
		return sms.SendResult{State: sms.SubmissionRejected}, sms.ErrInvalidMessage
	}

	response, err := p.client.send(ctx, sendPayload{
		Recipient:    []string{normalizeRecipient(message.To)},
		Sender:       strings.TrimSpace(message.From),
		Message:      message.Text,
		IsSchedule:   false,
		ScheduleDate: "",
	})
	if err != nil {
		return sms.SendResult{State: sms.SubmissionUnknown}, err
	}

	code := responseCode(response.body.Code)
	result := sms.SendResult{ProviderMessageID: response.body.Summary.MessageID}
	if response.statusCode == http.StatusOK && strings.EqualFold(response.body.Status, "success") && code == "2000" {
		result.State = sms.SubmissionAccepted
		return result, nil
	}

	// mNotify's public quick-send documentation clearly defines the success
	// response but does not provide enough acceptance semantics for every error
	// response. Stay conservative: ambiguous failures are unknown, not safe
	// fallback signals.
	result.State = sms.SubmissionUnknown
	return result, &APIError{StatusCode: response.statusCode, Code: code, Message: response.body.Message}
}

func normalizeRecipient(recipient string) string {
	value := strings.TrimSpace(recipient)
	return strings.TrimPrefix(value, "+")
}

func responseCode(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var number json.Number
	if err := json.Unmarshal(raw, &number); err == nil {
		return number.String()
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		if _, err := strconv.Atoi(text); err == nil {
			return text
		}
		return text
	}
	return ""
}
