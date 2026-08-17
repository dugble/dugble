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
)

type client struct {
	token   string
	baseURL *url.URL
	http    *http.Client
}

type sendPayload struct {
	To      string `json:"to"`
	From    string `json:"from"`
	Message string `json:"message"`
}

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

type smsResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		MessageID string `json:"messageId"`
		Status    string `json:"status"`
	} `json:"data"`
}

type senderIDResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type rawResponse struct {
	statusCode int
	body       smsResponse
}

type senderIDRawResponse struct {
	statusCode int
	body       senderIDResponse
}

func (c *client) send(ctx context.Context, payload sendPayload) (rawResponse, error) {
	return c.do(ctx, http.MethodPost, "/v1/sms/send", payload)
}

func (c *client) status(ctx context.Context, messageID string) (rawResponse, error) {
	return c.do(ctx, http.MethodGet, "/v1/sms/status/"+messageID, nil)
}

func (c *client) createSenderID(ctx context.Context, payload createSenderIDPayload) (senderIDRawResponse, error) {
	return c.senderIDRequest(ctx, http.MethodPost, "/v1/sender-ids", payload)
}

func (c *client) senderIDs(ctx context.Context) (senderIDRawResponse, error) {
	return c.senderIDRequest(ctx, http.MethodGet, "/v1/sender-ids", nil)
}

func (c *client) do(ctx context.Context, method, path string, payload any) (rawResponse, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return rawResponse{}, fmt.Errorf("encode Sendexa request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return rawResponse{}, fmt.Errorf("create Sendexa request: %w", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Basic "+c.token)

	response, err := c.http.Do(req)
	if err != nil {
		return rawResponse{}, fmt.Errorf("send Sendexa request: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	data, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	parsed := smsResponse{}
	if len(data) != 0 {
		if err := json.Unmarshal(data, &parsed); err != nil {
			return rawResponse{statusCode: response.StatusCode}, fmt.Errorf("decode Sendexa response: %w", err)
		}
	}

	raw := rawResponse{statusCode: response.StatusCode, body: parsed}
	if readErr != nil {
		return raw, fmt.Errorf("read Sendexa response: %w", readErr)
	}
	return raw, nil
}

func (c *client) senderIDRequest(ctx context.Context, method, path string, payload any) (senderIDRawResponse, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return senderIDRawResponse{}, fmt.Errorf("encode Sendexa sender ID request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return senderIDRawResponse{}, fmt.Errorf("create Sendexa sender ID request: %w", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Basic "+c.token)

	response, err := c.http.Do(req)
	if err != nil {
		return senderIDRawResponse{}, fmt.Errorf("send Sendexa sender ID request: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	data, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	parsed := senderIDResponse{}
	if len(data) != 0 {
		if err := json.Unmarshal(data, &parsed); err != nil {
			return senderIDRawResponse{statusCode: response.StatusCode}, fmt.Errorf("decode Sendexa sender ID response: %w", err)
		}
	}

	raw := senderIDRawResponse{statusCode: response.StatusCode, body: parsed}
	if readErr != nil {
		return raw, fmt.Errorf("read Sendexa sender ID response: %w", readErr)
	}
	return raw, nil
}
