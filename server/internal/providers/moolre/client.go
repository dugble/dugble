package moolre

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

const defaultBaseURL = "https://api.moolre.com"

type client struct {
	vasKey  string
	baseURL *url.URL
	http    *http.Client
}

type messagePayload struct {
	Recipient string `json:"recipient"`
	Message   string `json:"message"`
	Ref       string `json:"ref,omitempty"`
}

type sendPayload struct {
	Type     int              `json:"type"`
	SenderID string           `json:"senderid"`
	Messages []messagePayload `json:"messages"`
}

type createSenderIDPayload struct {
	Type      int               `json:"type"`
	SenderIDs []senderIDPayload `json:"senderids"`
}

type senderIDPayload struct {
	SenderID string `json:"senderid"`
}

type senderIDStatusPayload struct {
	Type     int    `json:"type"`
	SenderID string `json:"senderid"`
}

type smsStatusPayload struct {
	Type int      `json:"type"`
	Ref  []string `json:"ref"`
}

type responseBody struct {
	Status  int             `json:"status"`
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	Go      json.RawMessage `json:"go"`
}

type rawResponse struct {
	statusCode int
	body       responseBody
	rawBody    []byte
}

func newClient(config Config, baseURL string) (*client, error) {
	vasKey := strings.TrimSpace(config.VASKey)
	if vasKey == "" {
		return nil, fmt.Errorf("%w: VAS key is required", ErrInvalidConfig)
	}

	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("%w: invalid base URL", ErrInvalidConfig)
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &client{vasKey: vasKey, baseURL: parsed, http: httpClient}, nil
}

func (c *client) send(ctx context.Context, payload sendPayload) (rawResponse, error) {
	return c.post(ctx, "/open/sms/send", payload)
}

func (c *client) query(ctx context.Context, payload createSenderIDPayload) (rawResponse, error) {
	return c.post(ctx, "/open/sms/query", payload)
}

func (c *client) status(ctx context.Context, payload any) (rawResponse, error) {
	return c.post(ctx, "/open/sms/status", payload)
}

func (c *client) post(ctx context.Context, path string, payload any) (rawResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return rawResponse{}, fmt.Errorf("encode Moolre request: %w", err)
	}

	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return rawResponse{}, fmt.Errorf("create Moolre request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-API-VASKEY", c.vasKey)

	response, err := c.http.Do(req)
	if err != nil {
		return rawResponse{}, fmt.Errorf("send Moolre request: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	data, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	parsed := responseBody{}
	_ = json.Unmarshal(data, &parsed)
	raw := rawResponse{statusCode: response.StatusCode, body: parsed, rawBody: data}
	if readErr != nil {
		return raw, fmt.Errorf("read Moolre response: %w", readErr)
	}
	return raw, nil
}
