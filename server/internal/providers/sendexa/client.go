package sendexa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

type smsResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		MessageID string `json:"messageId"`
		Status    string `json:"status"`
	} `json:"data"`
}

type rawResponse struct {
	statusCode int
	body       smsResponse
}

func (c *client) send(ctx context.Context, payload sendPayload) (rawResponse, error) {
	return c.do(ctx, http.MethodPost, "/v1/sms/send", payload)
}

func (c *client) status(ctx context.Context, messageID string) (rawResponse, error) {
	return c.do(ctx, http.MethodGet, "/v1/sms/status/"+messageID, nil)
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

	endpoint := c.baseURL.ResolveReference(&url.URL{Path: path})
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
