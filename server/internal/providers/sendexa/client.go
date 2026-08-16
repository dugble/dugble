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

type sendResponse struct {
	Data struct {
		MessageID string `json:"messageId"`
		Delivery  struct {
			Status string `json:"status"`
		} `json:"delivery"`
	} `json:"data"`
}

type rawResponse struct {
	statusCode int
	body       sendResponse
}

func (c *client) send(ctx context.Context, payload sendPayload) (rawResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return rawResponse{}, fmt.Errorf("encode Sendexa request: %w", err)
	}

	endpoint := c.baseURL.ResolveReference(&url.URL{Path: "/v1/sms/send"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return rawResponse{}, fmt.Errorf("create Sendexa request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+c.token)

	response, err := c.http.Do(req)
	if err != nil {
		return rawResponse{}, fmt.Errorf("send Sendexa request: %w", err)
	}
	defer response.Body.Close()

	data, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return rawResponse{statusCode: response.StatusCode}, fmt.Errorf("read Sendexa response: %w", err)
	}

	parsed := sendResponse{}
	if len(data) != 0 {
		if err := json.Unmarshal(data, &parsed); err != nil {
			return rawResponse{statusCode: response.StatusCode}, fmt.Errorf("decode Sendexa response: %w", err)
		}
	}

	return rawResponse{statusCode: response.StatusCode, body: parsed}, nil
}
