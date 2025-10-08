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

// RequestBuilder builds requests for the SDK
type RequestBuilder struct {
	// Request parameters
	client *Client
	ctx    context.Context
	method string
	path   string
	in     any
	out    any

	// Authentication options
	apiKey       string
	clientID     string
	clientSecret string
}

// WithApiKey sets the API key for the request
func (rb *RequestBuilder) WithApiKey(apiKey string) *RequestBuilder {
	rb.apiKey = apiKey
	return rb
}

// WithClientCredentials sets the client credentials for the request
func (rb *RequestBuilder) WithClientCredentials(id, secret string) *RequestBuilder {
	rb.clientID = id
	rb.clientSecret = secret
	return rb
}

// doJSON is a helper to perform JSON requests to the backend
func (rb *RequestBuilder) doJSON() error {
	// Create request body if input is provided
	var body io.Reader
	if rb.in != nil {
		b, err := json.Marshal(rb.in)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(b)
	}

	// Create the request
	req, err := http.NewRequestWithContext(rb.ctx, rb.method, rb.client.baseURL+rb.path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// Set authentication headers
	if rb.apiKey != "" {
		req.Header.Set("X-API-KEY", rb.apiKey)
	}
	if rb.clientID != "" && rb.clientSecret != "" {
		req.Header.Set("X-CLIENT-ID", rb.clientID)
		req.Header.Set("X-CLIENT-SECRET", rb.clientSecret)
	}

	// Perform the request
	resp, err := rb.client.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// On error, read body and return error
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("[BACKEND]: backend '%s %s' failed: %d: %s", rb.method, rb.path, resp.StatusCode, string(b))
	}

	// If no output expected, return early
	if rb.out == nil {
		return nil
	}

	// Decode the response body into the output struct
	dec := json.NewDecoder(resp.Body)
	return dec.Decode(rb.out)
}

// Client wraps calls to the AI assistant backend
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new client for the AI assistant backend
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

// NewRequest creates a new request builder using a provided client
func (c *Client) NewRequest(ctx context.Context, method, path string, in, out any) *RequestBuilder {
	return &RequestBuilder{
		client: c,
		ctx:    ctx,
		method: method,
		path:   path,
		in:     in,
		out:    out,
	}
}
