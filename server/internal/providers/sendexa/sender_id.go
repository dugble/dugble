package sendexa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	provider "github.com/dugble/dugble/server/internal/providers"
)

const alphanumericSenderIDType = "ALPHANUMERIC"

type createSenderIDPayload struct {
	Name    string `json:"name"`
	UseCase string `json:"useCase"`
	Type    string `json:"type"`
}

type senderIDRecord struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type senderIDEnvelope struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// CreateSenderID submits an alphanumeric Sender ID registration request to Sendexa.
func (p *Provider) CreateSenderID(ctx context.Context, request provider.CreateSenderIDRequest) (provider.CreateSenderIDResult, error) {
	senderID := strings.TrimSpace(request.SenderID)
	useCase := strings.TrimSpace(request.UseCase)
	result := provider.CreateSenderIDResult{SenderID: senderID, Status: provider.SenderIDUnknown}
	if p == nil || p.client == nil {
		return result, ErrInvalidConfig
	}
	if senderID == "" || len(senderID) > p.Capabilities().MaxSenderIDLength || useCase == "" {
		return result, ErrInvalidRequest
	}

	response, statusCode, err := p.client.senderIDRequest(ctx, http.MethodPost, "/v1/sender-ids", createSenderIDPayload{
		Name:    senderID,
		UseCase: useCase,
		Type:    alphanumericSenderIDType,
	})
	if err != nil {
		return result, err
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices || !response.Success {
		return result, senderIDAPIError(statusCode, response.Message, "")
	}

	record, err := decodeSenderIDRecord(response.Data)
	if err != nil {
		return result, err
	}
	if strings.TrimSpace(record.Name) != "" {
		result.SenderID = strings.TrimSpace(record.Name)
	}
	result.ProviderReference = strings.TrimSpace(record.ID)
	result.Status = normalizeSenderIDStatus(record.Status)
	return result, nil
}

// CheckSenderIDStatus reconciles the current approval state of a Sendexa Sender ID.
// The list endpoint is searched by provider reference when available, otherwise by name.
func (p *Provider) CheckSenderIDStatus(ctx context.Context, request provider.SenderIDStatusRequest) (provider.SenderIDStatusResult, error) {
	senderID := strings.TrimSpace(request.SenderID)
	providerReference := strings.TrimSpace(request.ProviderReference)
	result := provider.SenderIDStatusResult{
		SenderID:          senderID,
		ProviderReference: providerReference,
		Status:            provider.SenderIDUnknown,
	}
	if p == nil || p.client == nil {
		return result, ErrInvalidConfig
	}
	if senderID == "" && providerReference == "" {
		return result, ErrInvalidRequest
	}

	response, statusCode, err := p.client.senderIDRequest(ctx, http.MethodGet, "/v1/sender-ids", nil)
	if err != nil {
		return result, err
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices || !response.Success {
		return result, senderIDAPIError(statusCode, response.Message, "")
	}

	var records []senderIDRecord
	if err := json.Unmarshal(response.Data, &records); err != nil {
		return result, fmt.Errorf("decode Sendexa sender IDs: %w", err)
	}
	for _, record := range records {
		id := strings.TrimSpace(record.ID)
		name := strings.TrimSpace(record.Name)
		if providerReference != "" {
			if id != providerReference {
				continue
			}
		} else if !strings.EqualFold(name, senderID) {
			continue
		}

		if name != "" {
			result.SenderID = name
		}
		if id != "" {
			result.ProviderReference = id
		}
		result.ProviderStatus = strings.TrimSpace(record.Status)
		result.Status = normalizeSenderIDStatus(record.Status)
		return result, nil
	}

	return result, fmt.Errorf("Sendexa sender ID not found")
}

func normalizeSenderIDStatus(status string) provider.SenderIDStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "approved", "active":
		return provider.SenderIDActive
	case "pending":
		return provider.SenderIDPending
	case "rejected":
		return provider.SenderIDRejected
	default:
		return provider.SenderIDUnknown
	}
}

func decodeSenderIDRecord(data json.RawMessage) (senderIDRecord, error) {
	var record senderIDRecord
	if len(data) == 0 || string(data) == "null" {
		return record, fmt.Errorf("decode Sendexa sender ID: response data is empty")
	}
	if err := json.Unmarshal(data, &record); err == nil && (record.ID != "" || record.Name != "" || record.Status != "") {
		return record, nil
	}

	var records []senderIDRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return record, fmt.Errorf("decode Sendexa sender ID: %w", err)
	}
	if len(records) == 0 {
		return record, fmt.Errorf("decode Sendexa sender ID: response data is empty")
	}
	return records[0], nil
}

func senderIDAPIError(statusCode int, message, status string) error {
	return &APIError{
		StatusCode: statusCode,
		Status:     strings.TrimSpace(status),
		Message:    strings.TrimSpace(message),
	}
}

func (c *client) senderIDRequest(ctx context.Context, method, path string, payload any) (senderIDEnvelope, int, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return senderIDEnvelope{}, 0, fmt.Errorf("encode Sendexa sender ID request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	endpoint := c.baseURL.ResolveReference(&url.URL{Path: path})
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return senderIDEnvelope{}, 0, fmt.Errorf("create Sendexa sender ID request: %w", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Basic "+c.token)

	response, err := c.http.Do(req)
	if err != nil {
		return senderIDEnvelope{}, 0, fmt.Errorf("send Sendexa sender ID request: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	data, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	parsed := senderIDEnvelope{}
	if len(data) != 0 {
		if err := json.Unmarshal(data, &parsed); err != nil {
			return senderIDEnvelope{}, response.StatusCode, fmt.Errorf("decode Sendexa sender ID response: %w", err)
		}
	}
	if readErr != nil {
		return parsed, response.StatusCode, fmt.Errorf("read Sendexa sender ID response: %w", readErr)
	}
	return parsed, response.StatusCode, nil
}

var _ provider.SenderIDCreator = (*Provider)(nil)
var _ provider.SenderIDStatusChecker = (*Provider)(nil)
