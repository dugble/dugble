package moolre

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/dugble/dugble/server/internal/relay/sms"
)

var ErrInvalidConfig = errors.New("invalid Moolre configuration")

// Config configures the Moolre SMS adapter.
type Config struct {
	VASKey     string
	BaseURL    string
	HTTPClient *http.Client
}

// Provider sends SMS through Moolre.
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

func (p *Provider) Send(ctx context.Context, message sms.Message) (sms.SendResult, error) {
	if p == nil || p.client == nil {
		return sms.SendResult{State: sms.SubmissionRejected}, ErrInvalidConfig
	}
	if strings.TrimSpace(message.From) == "" || !p.Capabilities().Supports(message) {
		return sms.SendResult{State: sms.SubmissionRejected}, sms.ErrInvalidMessage
	}

	response, requestErr := p.client.send(ctx, sendPayload{
		Type:     1,
		SenderID: strings.TrimSpace(message.From),
		Messages: []messagePayload{{
			Recipient: strings.TrimSpace(message.To),
			Message:   message.Text,
		}},
	})

	if response.statusCode == http.StatusOK && response.body.Status == 1 && strings.EqualFold(strings.TrimSpace(response.body.Code), "SMS01") {
		return sms.SendResult{State: sms.SubmissionAccepted}, nil
	}

	if isSafeToFallbackStatus(response.statusCode) {
		return sms.SendResult{State: sms.SubmissionRejected}, &APIError{
			StatusCode: response.statusCode,
			Code:       response.body.Code,
			Message:    response.body.Message,
		}
	}

	if requestErr != nil {
		return sms.SendResult{State: sms.SubmissionUnknown}, requestErr
	}

	return sms.SendResult{State: sms.SubmissionUnknown}, &APIError{
		StatusCode: response.statusCode,
		Code:       response.body.Code,
		Message:    response.body.Message,
	}
}

func isSafeToFallbackStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusBadRequest, http.StatusUnauthorized:
		return true
	default:
		return false
	}
}
