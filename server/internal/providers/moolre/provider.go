package moolre

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	provider "github.com/dugble/dugble/server/internal/providers"
	"github.com/dugble/dugble/server/internal/relay/sms"
)

var ErrInvalidConfig = errors.New("invalid Moolre configuration")
var ErrInvalidRequest = errors.New("invalid Moolre request")

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

// Send submits an SMS to Moolre. Submission classification remains a Relay
// concern: accepted stops, rejected may fall back, and unknown never falls back.
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
			Ref:       strings.TrimSpace(message.Reference),
		}},
	})

	if response.statusCode == http.StatusOK && response.body.Status == 1 && strings.EqualFold(strings.TrimSpace(response.body.Code), "SMS01") {
		return sms.SendResult{State: sms.SubmissionAccepted}, nil
	}

	if isSafeToFallbackStatus(response.statusCode) {
		return sms.SendResult{State: sms.SubmissionRejected}, apiError(response)
	}

	if requestErr != nil {
		return sms.SendResult{State: sms.SubmissionUnknown}, requestErr
	}

	return sms.SendResult{State: sms.SubmissionUnknown}, apiError(response)
}

// CreateSenderID submits a Sender ID registration request to Moolre.
func (p *Provider) CreateSenderID(ctx context.Context, request provider.CreateSenderIDRequest) (provider.CreateSenderIDResult, error) {
	senderID := strings.TrimSpace(request.SenderID)
	result := provider.CreateSenderIDResult{SenderID: senderID, Status: provider.SenderIDUnknown}
	if p == nil || p.client == nil {
		return result, ErrInvalidConfig
	}
	if senderID == "" || len(senderID) > 11 {
		return result, ErrInvalidRequest
	}

	response, err := p.client.query(ctx, createSenderIDPayload{
		Type:      3,
		SenderIDs: []senderIDPayload{{SenderID: senderID}},
	})
	if err != nil {
		return result, err
	}
	result.ProviderCode = response.body.Code
	if response.statusCode == http.StatusOK && response.body.Status == 1 && strings.EqualFold(strings.TrimSpace(response.body.Code), "ASMQ12") {
		result.Status = provider.SenderIDPending
		return result, nil
	}
	return result, apiError(response)
}

func isSafeToFallbackStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusBadRequest, http.StatusUnauthorized:
		return true
	default:
		return false
	}
}

func apiError(response rawResponse) error {
	return &APIError{
		StatusCode: response.statusCode,
		Code:       response.body.Code,
		Message:    response.body.Message,
	}
}

var _ provider.Sender = (*Provider)(nil)
var _ provider.SenderIDCreator = (*Provider)(nil)
var _ provider.SMSStatusChecker = (*Provider)(nil)
var _ provider.SenderIDStatusChecker = (*Provider)(nil)
