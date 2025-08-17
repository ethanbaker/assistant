package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// BackendClient wraps calls to the AI assistant backend
type BackendClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewBackendClient(baseURL string) *BackendClient {
	return &BackendClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

// Types for backend API

type Session struct {
	UUID string `json:"uuid"`
	// History and other fields may exist, but we only rely on UUID here
}

type CreateSessionRequest struct {
	// Placeholder for future params (agent id, etc.)
}

type MessageRequest struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type MessageResponse struct {
	Content string `json:"content"`
}

// Create a new session
func (c *BackendClient) CreateSession(ctx context.Context, req *CreateSessionRequest) (*Session, error) {
	var out Session
	if err := c.doJSON(ctx, http.MethodPost, "/api/agent/sessions", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get a session by UUID
func (c *BackendClient) GetSession(ctx context.Context, uuid string) (*Session, error) {
	var out Session
	path := fmt.Sprintf("/api/agent/sessions/%s", uuid)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Send a message to a session provided by UUID
func (c *BackendClient) SendMessage(ctx context.Context, uuid string, msg *MessageRequest) (*MessageResponse, error) {
	var out MessageResponse
	path := fmt.Sprintf("/api/agent/sessions/%s/message", uuid)
	if err := c.doJSON(ctx, http.MethodPost, path, msg, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete an existing session by UUID
func (c *BackendClient) DeleteSession(ctx context.Context, uuid string) error {
	path := fmt.Sprintf("/api/agent/sessions/%s", uuid)
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil)
}

// doJSON is a helper to perform JSON requests to the backend
func (c *BackendClient) doJSON(ctx context.Context, method, path string, in any, out any) error {
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
