package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ethanbaker/api/pkg/api_types"
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

// Create a new session
func (c *Client) CreateSession(ctx context.Context, req *CreateSessionRequest) (*Session, error) {
	path := "/api/agent/sessions"

	var out ApiResponse[Session]
	if err := c.doJSON(ctx, http.MethodPost, path, req, &out); err != nil {
		return nil, err
	}

	if out.Data.ID == "" {
		return nil, fmt.Errorf("no id returned")
	}

	return &out.Data, nil
}

// Get a session by UUID
func (c *Client) GetSession(ctx context.Context, uuid string) (*Session, error) {
	path := fmt.Sprintf("/api/agent/sessions/%s", uuid)

	var out ApiResponse[Session]
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}

	// Check for success
	switch out.Status {
	case api_types.StatusFail:
		return nil, fmt.Errorf("failed to get session: %s", out.Message)
	case api_types.StatusError:
		return nil, fmt.Errorf("error getting session (%s): %v", out.Message, out.Error)
	}

	// On success return data
	return &out.Data, nil
}

// Send a message to a session provided by UUID
func (c *Client) SendMessage(ctx context.Context, uuid string, msg *PostMessageRequest) (*PostMessageResponse, error) {
	path := fmt.Sprintf("/api/agent/sessions/%s/message", uuid)

	var out ApiResponse[PostMessageResponse]
	if err := c.doJSON(ctx, http.MethodPost, path, msg, &out); err != nil {
		return nil, err
	}

	return &out.Data, nil
}

// Delete an existing session by UUID
func (c *Client) DeleteSession(ctx context.Context, uuid string) error {
	path := fmt.Sprintf("/api/agent/sessions/%s", uuid)

	return c.doJSON(ctx, http.MethodDelete, path, nil, nil)
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
