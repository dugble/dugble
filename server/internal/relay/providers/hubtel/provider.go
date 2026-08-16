package hubtel

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/dugble/dugble/server/internal/relay/sms"
)

var ErrInvalidConfig = errors.New("invalid Hubtel configuration")

// Config configures the Hubtel Programmable SMS adapter.
type Config struct {
	ClientID     string
	ClientSecret string
	BaseURL      string
	HTTPClient   *http.Client
}

// Provider sends SMS through Hubtel Programmable SMS.
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

func (p *Provider) Name() string { return "hubtel" }

func (p *Provider) Capabilities() sms.Capabilities {
	return sms.Capabilities{
		AlphanumericSenderID:  true,
		MaxSenderIDLength:     11,
		RequiresE164Recipient: true,
	}
}

// HTTPError records a non-success response from Hubtel.
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	if strings.TrimSpace(e.Message) == "" {
		return fmt.Sprintf("Hubtel returned HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("Hubtel returned HTTP %d: %s", e.StatusCode, e.Message)
}

func (p *Provider) Send(ctx context.Context, message sms.Message) (sms.SendResult, error) {
	if p == nil || p.client == nil {
		return sms.SendResult{State: sms.SubmissionRejected}, ErrInvalidConfig
	}
	if !p.Capabilities().Supports(message) || strings.TrimSpace(message.From) == "" {
		return sms.SendResult{State: sms.SubmissionRejected}, sms.ErrInvalidMessage
	}

	response, err := p.client.send(ctx, sendRequest{
		From:    strings.TrimSpace(message.From),
		To:      strings.TrimSpace(message.To),
		Content: message.Text,
	})
	result := sms.SendResult{ProviderMessageID: response.body.Data.MessageID}

	switch response.statusCode {
	case http.StatusCreated:
		result.State = sms.SubmissionAccepted
		return result, nil
	case http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusNotFound,
		http.StatusTooManyRequests:
		result.State = sms.SubmissionRejected
		if err != nil {
			return result, &HTTPError{StatusCode: response.statusCode}
		}
		return result, &HTTPError{StatusCode: response.statusCode, Message: response.body.Message}
	default:
		result.State = sms.SubmissionUnknown
		if err != nil {
			return result, err
		}
		return result, &HTTPError{StatusCode: response.statusCode, Message: response.body.Message}
	}
}
