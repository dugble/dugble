package mnotify

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
	apiKey  string
	baseURL *url.URL
	http    *http.Client
}

type sendPayload struct {
	Recipient    []string `json:"recipient"`
	Sender       string   `json:"sender"`
	Message      string   `json:"message"`
	IsSchedule   bool     `json:"is_schedule"`
	ScheduleDate string   `json:"schedule_date"`
}

type sendResponse struct {
	Status  string          `json:"status"`
	Code    json.RawMessage `json:"code"`
	Message string          `json:"message"`
	Summary struct {
		MessageID string `json:"message_id"`
	} `json:"summary"`
}

type rawResponse struct {
	statusCode int
	body       sendResponse
}

func (c *client) send(ctx context.Context, payload sendPayload) (rawResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return rawResponse{}, fmt.Errorf("encode mNotify request: %w", err)
	}

	endpoint := *c.baseURL
	query := endpoint.Query()
	query.Set("key", c.apiKey)
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return rawResponse{}, fmt.Errorf("create mNotify request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	response, err := c.http.Do(req)
	if err != nil {
		return rawResponse{}, fmt.Errorf("send mNotify request: %w", err)
	}
	defer response.Body.Close()

	data, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return rawResponse{statusCode: response.StatusCode}, fmt.Errorf("read mNotify response: %w", err)
	}

	parsed := sendResponse{}
	if len(data) != 0 {
		if err := json.Unmarshal(data, &parsed); err != nil {
			return rawResponse{statusCode: response.StatusCode}, fmt.Errorf("decode mNotify response: %w", err)
		}
	}

	return rawResponse{statusCode: response.StatusCode, body: parsed}, nil
}
