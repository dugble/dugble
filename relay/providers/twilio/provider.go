package twilio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/dugble/relay/sms"
)

const defaultBaseURL = "https://api.twilio.com/2010-04-01"

var ErrInvalidConfig = errors.New("invalid Twilio configuration")

// Config configures the Twilio Programmable Messaging adapter.
// APIKey/APISecret are preferred for production. AuthToken is supported for
// local use with AccountSID as the Basic Auth username.
type Config struct {
	AccountSID string
	APIKey     string
	APISecret  string
	AuthToken  string
	BaseURL    string
	HTTPClient *http.Client
}

// Provider sends SMS through Twilio Programmable Messaging.
type Provider struct {
	client *client
}

func New(config Config) (*Provider, error) {
	accountSID := strings.TrimSpace(config.AccountSID)
	if accountSID == "" {
		return nil, fmt.Errorf("%w: account SID is required", ErrInvalidConfig)
	}

	apiKey := strings.TrimSpace(config.APIKey)
	apiSecret := strings.TrimSpace(config.APISecret)
	authToken := strings.TrimSpace(config.AuthToken)

	username := ""
	password := ""
	switch {
	case apiKey != "" || apiSecret != "":
		if apiKey == "" || apiSecret == "" {
			return nil, fmt.Errorf("%w: API key and API secret must be provided together", ErrInvalidConfig)
		}
		username = apiKey
		password = apiSecret
	case authToken != "":
		username = accountSID
		password = authToken
	default:
		return nil, fmt.Errorf("%w: API credentials or auth token are required", ErrInvalidConfig)
	}

	base := strings.TrimSpace(config.BaseURL)
	if base == "" {
		base = defaultBaseURL
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("%w: invalid base URL", ErrInvalidConfig)
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Provider{client: &client{
		accountSID: accountSID,
		username:   username,
		password:   password,
		baseURL:    parsed,
		http:       httpClient,
	}}, nil
}

func (p *Provider) Name() string { return "twilio" }

func (p *Provider) Capabilities() sms.Capabilities {
	return sms.Capabilities{
		AlphanumericSenderID:  true,
		RequiresE164Recipient: true,
	}
}

type messageResponse struct {
	SID    string `json:"sid"`
	Status string `json:"status"`
}

type errorResponse struct {
	Code     int    `json:"code"`
	Message  string `json:"message"`
	MoreInfo string `json:"more_info"`
	Status   int    `json:"status"`
}

// APIError records a non-success response returned by Twilio.
type APIError struct {
	StatusCode int
	Code       int
	Message    string
}

func (e *APIError) Error() string {
	message := strings.TrimSpace(e.Message)
	if message == "" {
		message = http.StatusText(e.StatusCode)
	}
	if e.Code != 0 {
		return fmt.Sprintf("Twilio returned HTTP %d code %d: %s", e.StatusCode, e.Code, message)
	}
	return fmt.Sprintf("Twilio returned HTTP %d: %s", e.StatusCode, message)
}

func (p *Provider) Send(ctx context.Context, message sms.Message) (sms.SendResult, error) {
	if p == nil || p.client == nil {
		return sms.SendResult{State: sms.SubmissionRejected}, ErrInvalidConfig
	}
	if strings.TrimSpace(message.From) == "" || !p.Capabilities().Supports(message) {
		return sms.SendResult{State: sms.SubmissionRejected}, sms.ErrInvalidMessage
	}

	response, requestErr := p.client.send(ctx, sendPayload{
		To:   strings.TrimSpace(message.To),
		From: strings.TrimSpace(message.From),
		Body: message.Text,
	})

	// A 201 means Twilio created the Message resource. Even if the response body
	// cannot be read or decoded, falling back could duplicate the SMS.
	if response.statusCode == http.StatusCreated {
		result := sms.SendResult{State: sms.SubmissionAccepted}
		if len(response.body) != 0 {
			var parsed messageResponse
			if json.Unmarshal(response.body, &parsed) == nil {
				result.ProviderMessageID = parsed.SID
			}
		}
		return result, nil
	}

	if isSafeToFallbackStatus(response.statusCode) {
		parsed := parseError(response.body)
		return sms.SendResult{State: sms.SubmissionRejected}, &APIError{
			StatusCode: response.statusCode,
			Code:       parsed.Code,
			Message:    parsed.Message,
		}
	}

	if requestErr != nil {
		return sms.SendResult{State: sms.SubmissionUnknown}, requestErr
	}

	parsed := parseError(response.body)
	return sms.SendResult{State: sms.SubmissionUnknown}, &APIError{
		StatusCode: response.statusCode,
		Code:       parsed.Code,
		Message:    parsed.Message,
	}
}

// isSafeToFallbackStatus contains only response statuses that establish that
// Twilio did not create/process the request. Timeout-like or otherwise
// ambiguous 4xx responses intentionally remain unknown.
func isSafeToFallbackStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusMethodNotAllowed,
		http.StatusRequestEntityTooLarge,
		http.StatusUnsupportedMediaType,
		http.StatusUnprocessableEntity,
		http.StatusTooManyRequests:
		return true
	default:
		return false
	}
}

func parseError(body []byte) errorResponse {
	var parsed errorResponse
	_ = json.Unmarshal(body, &parsed)
	return parsed
}
