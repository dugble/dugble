package moolre

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	provider "github.com/dugble/dugble/server/internal/providers"
	relaysenderid "github.com/dugble/dugble/server/internal/relay/senderid"
)

func (p *Provider) CheckSMSStatus(ctx context.Context, request provider.SMSStatusRequest) (provider.SMSStatusResult, error) {
	reference := strings.TrimSpace(request.Reference)
	result := provider.SMSStatusResult{Reference: reference, ProviderMessageID: strings.TrimSpace(request.ProviderMessageID), Status: provider.SMSUnknown}
	if p == nil || p.client == nil { return result, ErrInvalidConfig }
	if reference == "" { return result, ErrInvalidRequest }

	response, err := p.client.status(ctx, smsStatusPayload{Type: 5, Ref: []string{reference}})
	if err != nil { return result, err }
	result.ProviderCode = response.body.Code
	if response.statusCode != http.StatusOK || response.body.Status != 1 || !strings.EqualFold(strings.TrimSpace(response.body.Code), "ASMQ10") { return result, apiError(response) }

	var statuses []struct { Ref string `json:"ref"`; Status int `json:"status"` }
	if err := json.Unmarshal(response.body.Data, &statuses); err != nil { return result, fmt.Errorf("decode Moolre SMS status: %w", err) }
	for _, status := range statuses {
		if status.Ref != reference { continue }
		result.ProviderStatus = strconv.Itoa(status.Status)
		return result, nil
	}
	return result, fmt.Errorf("moolre SMS status did not include ref %q", reference)
}

func (p *Provider) CheckSenderIDStatus(ctx context.Context, request relaysenderid.StatusRequest) (relaysenderid.StatusResult, error) {
	senderID := strings.TrimSpace(request.Name)
	result := relaysenderid.StatusResult{Provider: p.Name(), Name: senderID, ProviderReference: strings.TrimSpace(request.ProviderReference), Status: relaysenderid.StatusUnknown}
	if p == nil || p.client == nil { return result, ErrInvalidConfig }
	if senderID == "" { return result, ErrInvalidRequest }

	response, err := p.client.status(ctx, senderIDStatusPayload{Type: 1, SenderID: senderID})
	if err != nil { return result, err }
	result.ProviderCode = response.body.Code
	if response.statusCode != http.StatusOK || response.body.Status != 1 || !strings.EqualFold(strings.TrimSpace(response.body.Code), "ASMQ01") { return result, apiError(response) }

	var data struct { SenderID string `json:"senderid"`; Approval string `json:"approval"`; Whitelisted bool `json:"whitelisted"` }
	if err := json.Unmarshal(response.body.Data, &data); err != nil { return result, fmt.Errorf("decode Moolre Sender ID status: %w", err) }
	if value := strings.TrimSpace(data.SenderID); value != "" { result.Name = value }
	result.ProviderStatus = strings.TrimSpace(data.Approval)
	switch strings.ToLower(result.ProviderStatus) {
	case "approved": result.Status = relaysenderid.StatusApproved
	case "pending": result.Status = relaysenderid.StatusPending
	case "rejected": result.Status = relaysenderid.StatusRejected
	default: result.Status = relaysenderid.StatusUnknown
	}
	return result, nil
}
