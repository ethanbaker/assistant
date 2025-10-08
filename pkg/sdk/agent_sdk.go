package sdk

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ethanbaker/api/pkg/api_types"
)

// Create a new session
func (c *Client) CreateSession(ctx context.Context, req *CreateSessionRequest) (*Session, error) {
	path := "/api/agent/sessions"

	var out ApiResponse[Session]
	if err := c.NewRequest(ctx, http.MethodPost, path, req, &out).WithApiKey(c.apiKey).doJSON(); err != nil {
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
	if err := c.NewRequest(ctx, http.MethodGet, path, nil, &out).WithApiKey(c.apiKey).doJSON(); err != nil {
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
	if err := c.NewRequest(ctx, http.MethodPost, path, msg, &out).WithApiKey(c.apiKey).doJSON(); err != nil {
		return nil, err
	}

	return &out.Data, nil
}

// Delete an existing session by UUID
func (c *Client) DeleteSession(ctx context.Context, uuid string) error {
	path := fmt.Sprintf("/api/agent/sessions/%s", uuid)

	return c.NewRequest(ctx, http.MethodDelete, path, nil, nil).WithApiKey(c.apiKey).doJSON()
}
