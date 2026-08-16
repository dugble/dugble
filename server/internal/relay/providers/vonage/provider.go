package vonage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/dugble/dugble/server/internal/relay/sms"
)

const defaultBaseURL = "https://api.nexmo.com/v1/messages"

var ErrInvalidConfig = errors.New("invalid Vonage configuration")

// Config configures the Vonage Messages API SMS adapter.
type Config struct {
	APIKey     string
	APISecret  string
	BaseURL    string
	HTTPClient *http.Client
}

// Provider sends SMS through the Vonage Messages API using account-level
// Basic authentication.
type Provider struct {
	client *client
}

func New(config Config) (*Provider, error) {
	apiKey := strings.TrimSpace(config.APIKey)
	apiSecret := strings.TrimSpace(config.APISecret)
	if apiKey == "" || apiSecret == "" {
		return nil, fmt.Errorf("%w: API key and API secret are required", ErrInvalidConfig)
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

	return &Provider{client: &client{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		baseURL:   parsed,
		http:      httpClient,
	}}, nil
}

func (p *Provider) Name() string { return "vonage" }

func (p *Provider) Capabilities() sms.Capabilities {
	return sms.Capabilities{
		AlphanumericSenderID:  true,
		RequiresE164Recipient: true,
	}
}

type sendResponse struct {
	MessageUUID string `json:"message_uuid"`
}

type problemResponse struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	Instance string `json:"instance"`
}

// APIError records a non-success response returned by Vonage.
type APIError struct {
	StatusCode int
	Type       string
	Title      string
	Detail     string
}

func (e *APIError) Error() string {
	message := strings.TrimSpace(e.Detail)
	if message == "" {
		message = strings.TrimSpace(e.Title)
	}
	if message == "" {
		message = http.StatusText(e.StatusCode)
	}
	return fmt.Sprintf("Vonage returned HTTP %d: %s", e.StatusCode, message)
}

func (p *Provider) Send(ctx context.Context, message sms.Message) (sms.SendResult, error) {
	if p == nil || p.client == nil {
		return sms.SendResult{State: sms.SubmissionRejected}, ErrInvalidConfig
	}
	if strings.TrimSpace(message.From) == "" || !p.Capabilities().Supports(message) {
		return sms.SendResult{State: sms.SubmissionRejected}, sms.ErrInvalidMessage
	}

	response, requestErr := p.client.send(ctx, sendPayload{
		MessageType: "text",
		Text:        message.Text,
		To:          normalizeNumber(message.To),
		From:        normalizeSender(message.From),
		Channel:     "sms",
	})

	// The Messages API documents HTTP 202 as successful acceptance. Once that
	// response is received, a malformed response body must not trigger fallback.
	if response.statusCode == http.StatusAccepted {
		result := sms.SendResult{State: sms.SubmissionAccepted}
		if len(response.body) != 0 {
			var parsed sendResponse
			if json.Unmarshal(response.body, &parsed) == nil {
				result.ProviderMessageID = parsed.MessageUUID
			}
		}
		return result, nil
	}

	if isSafeToFallbackStatus(response.statusCode) {
		problem := parseProblem(response.body)
		return sms.SendResult{State: sms.SubmissionRejected}, &APIError{
			StatusCode: response.statusCode,
			Type:       problem.Type,
			Title:      problem.Title,
			Detail:     problem.Detail,
		}
	}

	if requestErr != nil {
		return sms.SendResult{State: sms.SubmissionUnknown}, requestErr
	}

	problem := parseProblem(response.body)
	return sms.SendResult{State: sms.SubmissionUnknown}, &APIError{
		StatusCode: response.statusCode,
		Type:       problem.Type,
		Title:      problem.Title,
		Detail:     problem.Detail,
	}
}

// isSafeToFallbackStatus contains only response statuses that establish that
// the Messages API rejected the request before acceptance. Timeout-like or
// otherwise ambiguous 4xx responses intentionally remain unknown.
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

func normalizeNumber(number string) string {
	return strings.TrimPrefix(strings.TrimSpace(number), "+")
}

func normalizeSender(sender string) string {
	value := strings.TrimSpace(sender)
	if strings.HasPrefix(value, "+") {
		return strings.TrimPrefix(value, "+")
	}
	return value
}

func parseProblem(body []byte) problemResponse {
	var parsed problemResponse
	_ = json.Unmarshal(body, &parsed)
	return parsed
}
