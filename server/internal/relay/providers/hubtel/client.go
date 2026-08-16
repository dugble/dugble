package hubtel

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

const defaultBaseURL = "https://smsc.hubtel.com/v1/messages"

type client struct {
	clientID     string
	clientSecret string
	baseURL      *url.URL
	http         *http.Client
}

type sendRequest struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Content string `json:"content"`
}

type sendResponse struct {
	Message      string `json:"message"`
	ResponseCode string `json:"responseCode"`
	Data         struct {
		MessageID string `json:"messageId"`
		Status    string `json:"status"`
	} `json:"data"`
}

type rawResponse struct {
	statusCode int
	body       sendResponse
}

func newClient(config Config) (*client, error) {
	clientID := strings.TrimSpace(config.ClientID)
	clientSecret := strings.TrimSpace(config.ClientSecret)
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("%w: client ID and client secret are required", ErrInvalidConfig)
	}

	base := strings.TrimSpace(config.BaseURL)
	if base == "" {
		base = defaultBaseURL
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("%w: invalid base URL", ErrInvalidConfig)
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &client{
		clientID:     clientID,
		clientSecret: clientSecret,
		baseURL:      parsed,
		http:         httpClient,
	}, nil
}

func (c *client) send(ctx context.Context, payload sendRequest) (rawResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return rawResponse{}, fmt.Errorf("encode Hubtel request: %w", err)
	}

	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/send"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return rawResponse{}, fmt.Errorf("create Hubtel request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.clientID, c.clientSecret)

	response, err := c.http.Do(req)
	if err != nil {
		return rawResponse{}, fmt.Errorf("send Hubtel request: %w", err)
	}
	defer response.Body.Close()

	data, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	parsed := sendResponse{}
	_ = json.Unmarshal(data, &parsed)

	raw := rawResponse{statusCode: response.StatusCode, body: parsed}
	if readErr != nil {
		return raw, fmt.Errorf("read Hubtel response: %w", readErr)
	}
	return raw, nil
}
