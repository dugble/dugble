package resend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"strings"

	"github.com/dugble/dugble/server/internal/relay/email"
)

const defaultBaseURL = "https://api.resend.com"

var ErrInvalidConfig = errors.New("invalid Resend configuration")

type Config struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

type Provider struct { client *client }

func New(config Config) (*Provider, error) {
	apiKey := strings.TrimSpace(config.APIKey)
	if apiKey == "" { return nil, fmt.Errorf("%w: API key is required", ErrInvalidConfig) }
	base := strings.TrimSpace(config.BaseURL); if base == "" { base = defaultBaseURL }
	parsed, err := url.Parse(base); if err != nil || parsed.Scheme == "" || parsed.Host == "" { return nil, fmt.Errorf("%w: invalid base URL", ErrInvalidConfig) }
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	httpClient := config.HTTPClient; if httpClient == nil { httpClient = http.DefaultClient }
	return &Provider{client: &client{apiKey: apiKey, baseURL: parsed, http: httpClient}}, nil
}

func (p *Provider) Name() string { return "resend" }
func (p *Provider) Capabilities() email.Capabilities { return email.Capabilities{HTML: true, ReplyTo: true, MultipleRecipients: true, RequiresSubject: true, MaxRecipients: 50} }

type successResponse struct { ID string `json:"id"` }
type errorResponse struct { Name string `json:"name"`; Type string `json:"type"`; Message string `json:"message"` }

type APIError struct { StatusCode int; Type string; Message string }
func (e *APIError) Error() string { message := strings.TrimSpace(e.Message); if message == "" { message = http.StatusText(e.StatusCode) }; if e.Type != "" { return fmt.Sprintf("Resend returned HTTP %d %s: %s", e.StatusCode, e.Type, message) }; return fmt.Sprintf("Resend returned HTTP %d: %s", e.StatusCode, message) }

func (p *Provider) Send(ctx context.Context, message email.Message) (email.SendResult, error) {
	if p == nil || p.client == nil { return email.SendResult{State: email.SubmissionRejected}, ErrInvalidConfig }
	if !p.Capabilities().Supports(message) { return email.SendResult{State: email.SubmissionRejected}, errors.New("Resend does not support this email message") }
	payload := sendPayload{From: formatAddress(message.From), To: formatAddresses(message.To), Subject: message.Subject, HTML: message.HTML, Text: message.Text}
	if message.ReplyTo != nil { payload.ReplyTo = formatAddress(*message.ReplyTo) }
	response, requestErr := p.client.send(ctx, payload)
	if response.statusCode == http.StatusOK {
		result := email.SendResult{State: email.SubmissionAccepted}
		if len(response.body) != 0 { var parsed successResponse; if json.Unmarshal(response.body, &parsed) == nil { result.ProviderMessageID = parsed.ID } }
		return result, nil
	}
	parsed := parseError(response.body)
	if isSafeToFallback(response.statusCode, parsed.Type) { return email.SendResult{State: email.SubmissionRejected}, &APIError{StatusCode: response.statusCode, Type: parsed.Type, Message: parsed.Message} }
	if requestErr != nil { return email.SendResult{State: email.SubmissionUnknown}, requestErr }
	return email.SendResult{State: email.SubmissionUnknown}, &APIError{StatusCode: response.statusCode, Type: parsed.Type, Message: parsed.Message}
}

func isSafeToFallback(statusCode int, errorType string) bool {
	if statusCode == http.StatusConflict { return errorType == "invalid_idempotent_request" }
	switch statusCode { case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusMethodNotAllowed, http.StatusUnprocessableEntity, http.StatusTooManyRequests, http.StatusUnavailableForLegalReasons: return true; default: return false }
}

func parseError(body []byte) errorResponse { var parsed errorResponse; _ = json.Unmarshal(body, &parsed); if parsed.Type == "" { parsed.Type = parsed.Name }; return parsed }
func formatAddress(address email.Address) string { emailAddress := strings.TrimSpace(address.Email); name := strings.TrimSpace(address.Name); if name == "" { return emailAddress }; return (&mail.Address{Name: name, Address: emailAddress}).String() }
func formatAddresses(addresses []email.Address) []string { formatted := make([]string, 0, len(addresses)); for _, address := range addresses { formatted = append(formatted, formatAddress(address)) }; return formatted }
