package vonage

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
	apiKey    string
	apiSecret string
	baseURL   *url.URL
	http      *http.Client
}

type sendPayload struct {
	MessageType string `json:"message_type"`
	Text        string `json:"text"`
	To          string `json:"to"`
	From        string `json:"from"`
	Channel     string `json:"channel"`
}

type rawResponse struct {
	statusCode int
	body       []byte
}

func (c *client) send(ctx context.Context, payload sendPayload) (rawResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return rawResponse{}, fmt.Errorf("encode Vonage request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL.String(), bytes.NewReader(body))
	if err != nil {
		return rawResponse{}, fmt.Errorf("create Vonage request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(c.apiKey, c.apiSecret)

	response, err := c.http.Do(req)
	if err != nil {
		return rawResponse{}, fmt.Errorf("send Vonage request: %w", err)
	}
	defer response.Body.Close()

	data, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return rawResponse{statusCode: response.StatusCode}, fmt.Errorf("read Vonage response: %w", err)
	}
	return rawResponse{statusCode: response.StatusCode, body: data}, nil
}
