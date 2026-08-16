package postmark

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
	serverToken string
	baseURL     *url.URL
	http        *http.Client
}

type sendPayload struct {
	From          string `json:"From"`
	To            string `json:"To"`
	Subject       string `json:"Subject,omitempty"`
	HTMLBody      string `json:"HtmlBody,omitempty"`
	TextBody      string `json:"TextBody,omitempty"`
	ReplyTo       string `json:"ReplyTo,omitempty"`
	MessageStream string `json:"MessageStream"`
}

type rawResponse struct {
	statusCode int
	body       []byte
}

func (c *client) send(ctx context.Context, payload sendPayload) (rawResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return rawResponse{}, fmt.Errorf("encode Postmark request: %w", err)
	}

	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/email"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return rawResponse{}, fmt.Errorf("create Postmark request: %w", err)
	}
	req.Header.Set("X-Postmark-Server-Token", c.serverToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	response, err := c.http.Do(req)
	if err != nil {
		return rawResponse{}, fmt.Errorf("send Postmark request: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return rawResponse{statusCode: response.StatusCode}, fmt.Errorf("read Postmark response: %w", err)
	}
	return rawResponse{statusCode: response.StatusCode, body: responseBody}, nil
}
