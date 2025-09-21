package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client wraps calls to the AI assistant backend
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

// doJSON is a helper to perform JSON requests to the backend
func (c *Client) doJSON(ctx context.Context, method, path string, in any, out any) error {
	// Create request body if input is provided
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(b)
	}

	// Create the request
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", c.apiKey)

	// Perform the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// On error, read body and return error
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("[BACKEND]: backend '%s %s' failed: %d: %s", method, path, resp.StatusCode, string(b))
	}

	// If no output expected, return early
	if out == nil {
		return nil
	}

	// Decode the response body into the output struct
	dec := json.NewDecoder(resp.Body)
	return dec.Decode(out)
}
