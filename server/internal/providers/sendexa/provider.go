package sendexa

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/dugble/dugble/server/internal/relay/sms"
)

const defaultBaseURL = "https://api.sendexa.co"

var ErrInvalidConfig = errors.New("invalid Sendexa configuration")

// Config configures the Sendexa provider.
type Config struct {
	Token      string
	BaseURL    string
	HTTPClient *http.Client
}

// Provider owns communication with Sendexa.
type Provider struct {
	client *client
}

func New(config Config) (*Provider, error) {
	token := strings.TrimSpace(config.Token)
	if token == "" {
		return nil, fmt.Errorf("%w: token is required", ErrInvalidConfig)
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
}

func (e *APIError) Error() string {
	if e.Status != "" {
		return fmt.Sprintf("Sendexa returned HTTP %d with status %s", e.StatusCode, e.Status)
	}
	return fmt.Sprintf("Sendexa returned HTTP %d", e.StatusCode)
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

	status := strings.ToUpper(strings.TrimSpace(response.body.Data.Delivery.Status))
	result := sms.SendResult{ProviderMessageID: strings.TrimSpace(response.body.Data.MessageID)}
	if response.statusCode >= http.StatusOK && response.statusCode < http.StatusMultipleChoices && result.ProviderMessageID != "" && isAcceptedStatus(status) {
		result.State = sms.SubmissionAccepted
		return result, nil
	}

	result.State = sms.SubmissionUnknown
	return result, &APIError{StatusCode: response.statusCode, Status: status}
}

func isAcceptedStatus(status string) bool {
	switch status {
	case "PENDING", "SENT":
		return true
	default:
		return false
	}
}

func normalizeRecipient(recipient string) string {
	value := strings.TrimSpace(recipient)
	return strings.TrimPrefix(value, "+")
}
