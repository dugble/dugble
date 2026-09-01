package hubtel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dugble/dugble/server/internal/platform/config"
)

const (
	defaultBaseURL       = "https://payproxyapi.hubtel.com"
	defaultTxnStatusURL  = "https://api-txnstatus.hubtel.com"
	defaultClientTimeout = 30 * time.Second
	maxResponseBodyBytes = 1 << 20
)

type Client struct {
	APIID                 string
	APIKey                string
	MerchantAccountNumber string
	HTTPClient            *http.Client
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	if strings.TrimSpace(e.Body) == "" {
		return fmt.Sprintf("hubtel api returned status %d", e.StatusCode)
	}
	return fmt.Sprintf("hubtel api returned status %d: %s", e.StatusCode, e.Body)
}

func NewClient(cfg config.HubtelConfig) *Client {
	return &Client{
		APIID:                 strings.TrimSpace(cfg.ClientID),
		APIKey:                strings.TrimSpace(cfg.ClientSecret),
		MerchantAccountNumber: strings.TrimSpace(cfg.MerchantAccountNumber),
		HTTPClient:            &http.Client{Timeout: defaultClientTimeout},
	}
}

func (c *Client) InitiateCheckout(ctx context.Context, req InitiateCheckoutRequest) (InitiateCheckoutResponse, error) {
	if req.MerchantAccountNumber == "" {
		req.MerchantAccountNumber = c.MerchantAccountNumber
	}
	var result InitiateCheckoutResponse
	if err := c.doRequest(ctx, http.MethodPost, defaultBaseURL, "/items/initiate", req, &result); err != nil {
		return InitiateCheckoutResponse{}, err
	}
	return result, nil
}

func (c *Client) CheckTransactionStatus(ctx context.Context, clientReference string) (TransactionStatusResponse, error) {
	var result TransactionStatusResponse
	path := "/transactions/" + c.MerchantAccountNumber + "/status?clientReference=" + clientReference
	if err := c.doRequest(ctx, http.MethodGet, defaultTxnStatusURL, path, nil, &result); err != nil {
		return TransactionStatusResponse{}, err
	}
	return result, nil
}

func (c *Client) doRequest(ctx context.Context, method string, baseURL string, path string, payload any, result any) error {
	if c == nil {
		return errors.New("hubtel client is nil")
	}
	if strings.TrimSpace(baseURL) == "" {
		return errors.New("hubtel base URL is required")
	}
	if strings.TrimSpace(c.APIID) == "" || strings.TrimSpace(c.APIKey) == "" {
		return errors.New("hubtel API ID and API key are required")
	}
	if c.HTTPClient == nil {
		return errors.New("hubtel HTTP client is required")
	}
	var body io.Reader = http.NoBody
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode hubtel request: %w", err)
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(baseURL, "/")+"/"+strings.TrimLeft(path, "/"), body)
	if err != nil {
		return fmt.Errorf("create hubtel request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.APIID, c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("send hubtel request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))
	if err != nil {
		return fmt.Errorf("read hubtel response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return &APIError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(responseBody))}
	}
	if result == nil || len(bytes.TrimSpace(responseBody)) == 0 {
		return nil
	}
	if err := json.Unmarshal(responseBody, result); err != nil {
		return fmt.Errorf("decode hubtel response: %w", err)
	}
	return nil
}
