package netfetch

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

const defaultTimeout = 10 * time.Second

// Client wraps HTTP operations for internet data fetches.
type Client struct {
	httpClient *resty.Client
}

// NewClient creates a fetch client with the provided timeout.
// If timeout is zero or negative, a safe default is used.
func NewClient(timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	rc := resty.New().
		SetTimeout(timeout).
		SetHeader("Accept", "application/json")

	return &Client{httpClient: rc}
}

// GetJSON fetches URL data and decodes a JSON response into out.
func (c *Client) GetJSON(ctx context.Context, rawURL string, headers map[string]string, query map[string]string, out any) (int, error) {
	trimmedURL := strings.TrimSpace(rawURL)
	if trimmedURL == "" {
		return 0, fmt.Errorf("url is required")
	}
	if out == nil {
		return 0, fmt.Errorf("out target is required")
	}

	req := c.httpClient.R().SetContext(ctx).SetResult(out)
	for key, value := range headers {
		k := strings.TrimSpace(key)
		v := strings.TrimSpace(value)
		if k == "" || v == "" {
			continue
		}
		req.SetHeader(k, v)
	}
	for key, value := range query {
		k := strings.TrimSpace(key)
		v := strings.TrimSpace(value)
		if k == "" || v == "" {
			continue
		}
		req.SetQueryParam(k, v)
	}

	resp, err := req.Get(trimmedURL)
	if err != nil {
		return 0, fmt.Errorf("get %s: %w", trimmedURL, err)
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return resp.StatusCode(), fmt.Errorf("get %s failed: status=%d", trimmedURL, resp.StatusCode())
	}

	return resp.StatusCode(), nil
}
