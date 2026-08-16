package twilio

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type client struct {
	accountSID string
	username   string
	password   string
	baseURL    *url.URL
	http       *http.Client
}

type sendPayload struct {
	To   string
	From string
	Body string
}

type rawResponse struct {
	statusCode int
	body       []byte
}

func (c *client) send(ctx context.Context, payload sendPayload) (rawResponse, error) {
	form := url.Values{}
	form.Set("To", payload.To)
	form.Set("From", payload.From)
	form.Set("Body", payload.Body)

	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/Accounts/" + url.PathEscape(c.accountSID) + "/Messages.json"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), strings.NewReader(form.Encode()))
	if err != nil {
		return rawResponse{}, fmt.Errorf("create Twilio request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(c.username, c.password)

	response, err := c.http.Do(req)
	if err != nil {
		return rawResponse{}, fmt.Errorf("send Twilio request: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return rawResponse{statusCode: response.StatusCode}, fmt.Errorf("read Twilio response: %w", err)
	}
	return rawResponse{statusCode: response.StatusCode, body: body}, nil
}
