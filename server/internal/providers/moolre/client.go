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

const defaultBaseURL = "https://api.moolre.com/open/sms/send"

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

func newClient(config Config) (*client, error) {
	vasKey := strings.TrimSpace(config.VASKey)
	if vasKey == "" {
		return nil, fmt.Errorf("%w: VAS key is required", ErrInvalidConfig)
	}

	base := strings.TrimSpace(config.BaseURL)
	if base == "" {
		base = defaultBaseURL
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("%w: invalid base URL", ErrInvalidConfig)
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &client{vasKey: vasKey, baseURL: parsed, http: httpClient}, nil
}

func (c *client) send(ctx context.Context, payload sendPayload) (rawResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return rawResponse{}, fmt.Errorf("encode Moolre request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL.String(), bytes.NewReader(body))
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
	defer response.Body.Close()

	data, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	parsed := responseBody{}
	_ = json.Unmarshal(data, &parsed)
	raw := rawResponse{statusCode: response.StatusCode, body: parsed, rawBody: data}
	if readErr != nil {
		return raw, fmt.Errorf("read Moolre response: %w", readErr)
	}
	return raw, nil
}
