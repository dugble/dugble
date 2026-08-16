package moolre

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	provider "github.com/dugble/dugble/server/internal/providers"
)

// CheckSMSStatus reconciles an SMS previously submitted with a Moolre ref.
// Moolre documents numeric provider statuses but does not currently document
// their semantic mapping, so the native value is preserved and the normalized
// status remains unknown until that mapping is authoritative.
func (p *Provider) CheckSMSStatus(ctx context.Context, request provider.SMSStatusRequest) (provider.SMSStatusResult, error) {
	reference := strings.TrimSpace(request.Reference)
	result := provider.SMSStatusResult{
		Reference:         reference,
		ProviderMessageID: strings.TrimSpace(request.ProviderMessageID),
		Status:            provider.SMSUnknown,
	}
	if p == nil || p.client == nil {
		return result, ErrInvalidConfig
	}
	if reference == "" {
		return result, ErrInvalidRequest
	}

	response, err := p.client.status(ctx, smsStatusPayload{Type: 5, Ref: []string{reference}})
	if err != nil {
		return result, err
	}
	result.ProviderCode = response.body.Code
	if response.statusCode != http.StatusOK || response.body.Status != 1 || !strings.EqualFold(strings.TrimSpace(response.body.Code), "ASMQ10") {
		return result, apiError(response)
	}

	var statuses []struct {
		Ref    string `json:"ref"`
		Status int    `json:"status"`
	}
	if err := json.Unmarshal(response.body.Data, &statuses); err != nil {
		return result, fmt.Errorf("decode Moolre SMS status: %w", err)
	}
	for _, status := range statuses {
		if status.Ref != reference {
			continue
		}
		result.ProviderStatus = strconv.Itoa(status.Status)
		return result, nil
	}
	return result, fmt.Errorf("moolre SMS status did not include ref %q", reference)
}

// CheckSenderIDStatus reconciles the approval state of a Moolre Sender ID.
func (p *Provider) CheckSenderIDStatus(ctx context.Context, request provider.SenderIDStatusRequest) (provider.SenderIDStatusResult, error) {
	senderID := strings.TrimSpace(request.SenderID)
	result := provider.SenderIDStatusResult{
		SenderID:          senderID,
		ProviderReference: strings.TrimSpace(request.ProviderReference),
		Status:            provider.SenderIDUnknown,
	}
	if p == nil || p.client == nil {
		return result, ErrInvalidConfig
	}
	if senderID == "" {
		return result, ErrInvalidRequest
	}

	response, err := p.client.status(ctx, senderIDStatusPayload{Type: 1, SenderID: senderID})
	if err != nil {
		return result, err
	}
	result.ProviderCode = response.body.Code
	if response.statusCode != http.StatusOK || response.body.Status != 1 || !strings.EqualFold(strings.TrimSpace(response.body.Code), "ASMQ01") {
		return result, apiError(response)
	}

	var data struct {
		SenderID    string `json:"senderid"`
		Approval    string `json:"approval"`
		Whitelisted bool   `json:"whitelisted"`
	}
	if err := json.Unmarshal(response.body.Data, &data); err != nil {
		return result, fmt.Errorf("decode Moolre Sender ID status: %w", err)
	}
	if returned := strings.TrimSpace(data.SenderID); returned != "" && !strings.EqualFold(returned, senderID) {
		return result, fmt.Errorf("moolre Sender ID status returned %q for %q", returned, senderID)
	}
	result.ProviderStatus = strings.TrimSpace(data.Approval)
	result.Whitelisted = data.Whitelisted
	switch strings.ToLower(result.ProviderStatus) {
	case "approved":
		result.Status = provider.SenderIDActive
	case "pending":
		result.Status = provider.SenderIDPending
	case "rejected":
		result.Status = provider.SenderIDRejected
	default:
		result.Status = provider.SenderIDUnknown
	}
	return result, nil
}
