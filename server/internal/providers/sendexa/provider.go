package sendexa

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	provider "github.com/dugble/dugble/server/internal/providers"
	"github.com/dugble/dugble/server/internal/relay/sms"
)

const defaultBaseURL = "https://api.sendexa.co"

var ErrInvalidConfig = errors.New("invalid Sendexa configuration")
var ErrInvalidRequest = errors.New("invalid Sendexa request")

// Config configures the Sendexa provider.
type Config struct {
	Token      string
	HTTPClient *http.Client
}

// Provider owns communication with Sendexa.
type Provider struct {
	client *client
}

func New(config Config) (*Provider, error) {
	return newProvider(config, defaultBaseURL)
}

func newProvider(config Config, baseURL string) (*Provider, error) {
	token := strings.TrimSpace(config.Token)
	if token == "" {
		return nil, fmt.Errorf("%w: token is required", ErrInvalidConfig)
	}

	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("%w: invalid base URL", ErrInvalidConfig)
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Provider{client: &client{token: token, baseURL: parsed, http: httpClient}}, nil
}

func (p *Provider) Name() string { return "sendexa" }

func (p *Provider) Capabilities() sms.Capabilities {
	return sms.Capabilities{
		AlphanumericSenderID: true,
		MaxSenderIDLength:    11,
	}
}

// APIError records a non-success response returned by Sendexa.
type APIError struct {
	StatusCode int
	Status     string
	Message    string
}

func (e *APIError) Error() string {
	message := strings.TrimSpace(e.Message)
	if message == "" {
		message = http.StatusText(e.StatusCode)
	}
	if e.Status != "" {
		return fmt.Sprintf("Sendexa returned HTTP %d with status %s: %s", e.StatusCode, e.Status, message)
	}
	return fmt.Sprintf("Sendexa returned HTTP %d: %s", e.StatusCode, message)
}

// Send submits an SMS to Sendexa. Only a successful HTTP response with a
// Sendexa message ID and a known accepted submission status is accepted.
func (p *Provider) Send(ctx context.Context, message sms.Message) (sms.SendResult, error) {
	if p == nil || p.client == nil {
		return sms.SendResult{State: sms.SubmissionRejected}, ErrInvalidConfig
	}
	if strings.TrimSpace(message.From) == "" || !p.Capabilities().Supports(message) {
		return sms.SendResult{State: sms.SubmissionRejected}, sms.ErrInvalidMessage
	}

	response, err := p.client.send(ctx, sendPayload{
		To:      normalizeRecipient(message.To),
		From:    strings.TrimSpace(message.From),
		Message: message.Text,
	})
	if err != nil {
		return sms.SendResult{State: sms.SubmissionUnknown}, err
	}

	status := strings.ToLower(strings.TrimSpace(response.body.Data.Status))
	result := sms.SendResult{
		ProviderMessageID: strings.TrimSpace(response.body.Data.MessageID),
		ProviderStatus:    status,
	}
	if response.statusCode >= http.StatusOK && response.statusCode < http.StatusMultipleChoices && response.body.Success && result.ProviderMessageID != "" && isAcceptedStatus(status) {
		result.State = sms.SubmissionAccepted
		return result, nil
	}

	result.State = sms.SubmissionUnknown
	return result, apiError(response)
}

func isAcceptedStatus(status string) bool {
	switch status {
	case "queued", "sent":
		return true
	default:
		return false
	}
}

func normalizeRecipient(recipient string) string {
	value := strings.TrimSpace(recipient)
	return strings.TrimPrefix(value, "+")
}

func apiError(response rawResponse) error {
	return &APIError{
		StatusCode: response.statusCode,
		Status:     strings.TrimSpace(response.body.Data.Status),
		Message:    response.body.Message,
	}
}

var _ provider.Sender = (*Provider)(nil)
var _ provider.SMSStatusChecker = (*Provider)(nil)
