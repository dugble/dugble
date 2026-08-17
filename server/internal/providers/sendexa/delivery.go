package sendexa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	provider "github.com/dugble/dugble/server/internal/providers"
	relaysenderid "github.com/dugble/dugble/server/internal/relay/senderid"
)

func (p *Provider) CheckSMSStatus(ctx context.Context, request provider.SMSStatusRequest) (provider.SMSStatusResult, error) {
	messageID := strings.TrimSpace(request.ProviderMessageID)
	result := provider.SMSStatusResult{Reference: strings.TrimSpace(request.Reference), ProviderMessageID: messageID, Status: provider.SMSUnknown}
	if p == nil || p.client == nil { return result, ErrInvalidConfig }
	if messageID == "" { return result, ErrInvalidRequest }
	response, err := p.client.status(ctx, messageID)
	if err != nil { return result, err }
	if response.statusCode < http.StatusOK || response.statusCode >= http.StatusMultipleChoices || !response.body.Success { return result, apiError(response) }
	if returnedID := strings.TrimSpace(response.body.Data.MessageID); returnedID != "" { result.ProviderMessageID = returnedID }
	result.ProviderStatus = strings.TrimSpace(response.body.Data.Status)
	result.Status = normalizeSMSStatus(result.ProviderStatus)
	return result, nil
}

func (p *Provider) CheckSenderIDStatus(ctx context.Context, request relaysenderid.StatusRequest) (relaysenderid.StatusResult, error) {
	senderID := strings.TrimSpace(request.Name)
	providerReference := strings.TrimSpace(request.ProviderReference)
	result := relaysenderid.StatusResult{Provider: p.Name(), Name: senderID, ProviderReference: providerReference, Status: relaysenderid.StatusUnknown}
	if p == nil || p.client == nil { return result, ErrInvalidConfig }
	if senderID == "" && providerReference == "" { return result, ErrInvalidRequest }
	response, err := p.client.senderIDs(ctx)
	if err != nil { return result, err }
	if response.statusCode < http.StatusOK || response.statusCode >= http.StatusMultipleChoices || !response.body.Success { return result, senderIDAPIError(response.statusCode, response.body.Message, "") }
	var records []senderIDRecord
	if err := json.Unmarshal(response.body.Data, &records); err != nil { return result, fmt.Errorf("decode Sendexa sender IDs: %w", err) }
	for _, record := range records {
		id := strings.TrimSpace(record.ID)
		name := strings.TrimSpace(record.Name)
		if providerReference != "" { if id != providerReference { continue } } else if !strings.EqualFold(name, senderID) { continue }
		if name != "" { result.Name = name }
		if id != "" { result.ProviderReference = id }
		result.ProviderStatus = strings.TrimSpace(record.Status)
		result.Status = normalizeSenderIDStatus(record.Status)
		return result, nil
	}
	return result, fmt.Errorf("Sendexa sender ID not found")
}

func normalizeSMSStatus(status string) provider.SMSStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "queued", "sent": return provider.SMSPending
	case "delivered": return provider.SMSDelivered
	case "failed", "expired": return provider.SMSFailed
	default: return provider.SMSUnknown
	}
}

func normalizeSenderIDStatus(status string) relaysenderid.Status {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "approved", "active": return relaysenderid.StatusApproved
	case "pending": return relaysenderid.StatusPending
	case "rejected": return relaysenderid.StatusRejected
	default: return relaysenderid.StatusUnknown
	}
}
