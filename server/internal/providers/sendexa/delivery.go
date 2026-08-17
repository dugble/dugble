package sendexa

import (
	"context"
	"net/http"
	"strings"

	provider "github.com/dugble/dugble/server/internal/providers"
)

// CheckSMSStatus reconciles the delivery state of a Sendexa SMS using the
// provider message ID returned by Send.
func (p *Provider) CheckSMSStatus(ctx context.Context, request provider.SMSStatusRequest) (provider.SMSStatusResult, error) {
	messageID := strings.TrimSpace(request.ProviderMessageID)
	result := provider.SMSStatusResult{
		Reference:         strings.TrimSpace(request.Reference),
		ProviderMessageID: messageID,
		Status:            provider.SMSUnknown,
	}
	if p == nil || p.client == nil {
		return result, ErrInvalidConfig
	}
	if messageID == "" {
		return result, ErrInvalidRequest
	}

	response, err := p.client.status(ctx, messageID)
	if err != nil {
		return result, err
	}
	if response.statusCode < http.StatusOK || response.statusCode >= http.StatusMultipleChoices || !response.body.Success {
		return result, apiError(response)
	}

	if returnedID := strings.TrimSpace(response.body.Data.MessageID); returnedID != "" {
		result.ProviderMessageID = returnedID
	}
	result.ProviderStatus = strings.TrimSpace(response.body.Data.Status)
	result.Status = normalizeSMSStatus(result.ProviderStatus)
	return result, nil
}

func normalizeSMSStatus(status string) provider.SMSStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "queued", "sent":
		return provider.SMSPending
	case "delivered":
		return provider.SMSDelivered
	case "failed", "expired":
		return provider.SMSFailed
	default:
		return provider.SMSUnknown
	}
}
