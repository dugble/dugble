package postmark

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"strings"

	relaycore "github.com/dugble/dugble/server/internal/relay"
	"github.com/dugble/dugble/server/internal/relay/email"
)

const (
	defaultBaseURL       = "https://api.postmarkapp.com"
	defaultMessageStream = "outbound"
)

var ErrInvalidConfig = errors.New("invalid Postmark configuration")

type Config struct {
	ServerToken   string
	MessageStream string
	BaseURL       string
	HTTPClient    *http.Client
}

type Provider struct {
	client        *client
	messageStream string
}

func New(config Config) (*Provider, error) {
	serverToken := strings.TrimSpace(config.ServerToken)
	if serverToken == "" { return nil, fmt.Errorf("%w: server token is required", ErrInvalidConfig) }
	messageStream := strings.TrimSpace(config.MessageStream); if messageStream == "" { messageStream = defaultMessageStream }
	base := strings.TrimSpace(config.BaseURL); if base == "" { base = defaultBaseURL }
	parsed, err := url.Parse(base); if err != nil || parsed.Scheme == "" || parsed.Host == "" { return nil, fmt.Errorf("%w: invalid base URL", ErrInvalidConfig) }
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	httpClient := config.HTTPClient; if httpClient == nil { httpClient = http.DefaultClient }
	return &Provider{client: &client{serverToken: serverToken, baseURL: parsed, http: httpClient}, messageStream: messageStream}, nil
}

func (p *Provider) Name() string { return "postmark" }
func (p *Provider) Capabilities() email.Capabilities { return email.Capabilities{HTML: true, ReplyTo: true, MultipleRecipients: true, MaxRecipients: 50} }

type successResponse struct { MessageID string `json:"MessageID"` }
type errorResponse struct { ErrorCode int `json:"ErrorCode"`; Message string `json:"Message"` }
type APIError struct { StatusCode int; ErrorCode int; Message string }

func (e *APIError) Error() string { message := strings.TrimSpace(e.Message); if message == "" { message = http.StatusText(e.StatusCode) }; if e.ErrorCode != 0 { return fmt.Sprintf("Postmark returned HTTP %d code %d: %s", e.StatusCode, e.ErrorCode, message) }; return fmt.Sprintf("Postmark returned HTTP %d: %s", e.StatusCode, message) }

func (p *Provider) Send(ctx context.Context, message email.Message) (email.SendResult, error) {
	if p == nil || p.client == nil { return email.SendResult{State: email.SubmissionRejected}, ErrInvalidConfig }
	if !p.Capabilities().Supports(message) { return email.SendResult{State: email.SubmissionRejected}, relaycore.ErrInvalidMessage }
	payload := sendPayload{From: formatAddress(message.From), To: strings.Join(formatAddresses(message.To), ","), Subject: message.Subject, HTMLBody: message.HTML, TextBody: message.Text, MessageStream: p.messageStream}
	if message.ReplyTo != nil { payload.ReplyTo = formatAddress(*message.ReplyTo) }
	response, requestErr := p.client.send(ctx, payload)
	if response.statusCode == http.StatusOK {
		result := email.SendResult{State: email.SubmissionAccepted}
		if len(response.body) != 0 { var parsed successResponse; if json.Unmarshal(response.body, &parsed) == nil { result.ProviderMessageID = parsed.MessageID } }
		return result, nil
	}
	parsed := parseError(response.body)
	if isSafeToFallbackStatus(response.statusCode) { return email.SendResult{State: email.SubmissionRejected}, &APIError{StatusCode: response.statusCode, ErrorCode: parsed.ErrorCode, Message: parsed.Message} }
	if requestErr != nil { return email.SendResult{State: email.SubmissionUnknown}, requestErr }
	return email.SendResult{State: email.SubmissionUnknown}, &APIError{StatusCode: response.statusCode, ErrorCode: parsed.ErrorCode, Message: parsed.Message}
}

func isSafeToFallbackStatus(statusCode int) bool {
	switch statusCode { case http.StatusUnauthorized, http.StatusNotFound, http.StatusRequestEntityTooLarge, http.StatusUnsupportedMediaType, http.StatusUnprocessableEntity, http.StatusTooManyRequests: return true; default: return false }
}

func parseError(body []byte) errorResponse { var parsed errorResponse; _ = json.Unmarshal(body, &parsed); return parsed }
func formatAddress(address email.Address) string { emailAddress := strings.TrimSpace(address.Email); name := strings.TrimSpace(address.Name); if name == "" { return emailAddress }; return (&mail.Address{Name: name, Address: emailAddress}).String() }
func formatAddresses(addresses []email.Address) []string { formatted := make([]string, 0, len(addresses)); for _, address := range addresses { formatted = append(formatted, formatAddress(address)) }; return formatted }
