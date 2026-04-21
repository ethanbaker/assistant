package sdk

import (
	"context"
	"fmt"
	"net/http"
)

// CreateSession creates a new agent session
func (c *Client) CreateSession(ctx context.Context, req *CreateSessionRequest) (*Session, error) {
	path := "/api/internal/agent/sessions"

	var out SuccessResponse[Session]
	if err := c.NewRequest(ctx, http.MethodPost, path, req, &out).WithApiKey(c.apiKey).doJSON(); err != nil {
		return nil, err
	}

	if out.Data.ID == "" {
		return nil, fmt.Errorf("no id returned")
	}

	return &out.Data, nil
}

// GetSession retrieves an existing session by UUID
func (c *Client) GetSession(ctx context.Context, uuid string) (*Session, error) {
	path := fmt.Sprintf("/api/internal/agent/sessions/%s", uuid)

	var out SuccessResponse[Session]
	if err := c.NewRequest(ctx, http.MethodGet, path, nil, &out).WithApiKey(c.apiKey).doJSON(); err != nil {
		return nil, err
	}

	return &out.Data, nil
}

// SendMessage sends a message to a session and returns the agent response
func (c *Client) SendMessage(ctx context.Context, uuid string, msg *PostMessageRequest) (*PostMessageResponse, error) {
	path := fmt.Sprintf("/api/internal/agent/sessions/%s/message", uuid)

	var out SuccessResponse[PostMessageResponse]
	if err := c.NewRequest(ctx, http.MethodPost, path, msg, &out).WithApiKey(c.apiKey).doJSON(); err != nil {
		return nil, err
	}

	return &out.Data, nil
}

// DeleteSession deletes an existing session by UUID and returns the deleted session
func (c *Client) DeleteSession(ctx context.Context, uuid string) (*Session, error) {
	path := fmt.Sprintf("/api/internal/agent/sessions/%s", uuid)

	var out SuccessResponse[Session]
	if err := c.NewRequest(ctx, http.MethodDelete, path, nil, &out).WithApiKey(c.apiKey).doJSON(); err != nil {
		return nil, err
	}

	return &out.Data, nil
}

// AttachJobExecutionContext attaches outreach job execution context to an existing session
func (c *Client) AttachJobExecutionContext(ctx context.Context, uuid string, clientKey string, req *AttachJobExecutionContextRequest) error {
	path := fmt.Sprintf("/api/internal/agent/sessions/%s/context/job-execution", uuid)

	return c.NewRequest(ctx, http.MethodPost, path, req, nil).WithOutreachClientKey(clientKey).doJSON()
}
