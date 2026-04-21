package sdk

import (
	"context"
	"net/http"
)

// CreateJob creates a new outreach job (requires admin API key)
func (c *Client) CreateJob(ctx context.Context, req *CreateJobRequest) (*CreateJobResponse, error) {
	path := "/api/internal/outreach/jobs"

	var out SuccessResponse[CreateJobResponse]
	if err := c.NewRequest(ctx, http.MethodPost, path, req, &out).WithApiKey(c.apiKey).doJSON(); err != nil {
		return nil, err
	}

	return &out.Data, nil
}

// RegisterClient registers a new outreach client and returns its ID and API key (requires admin API key)
func (c *Client) RegisterClient(ctx context.Context, req *RegisterClientRequest) (*RegisterClientResponse, error) {
	path := "/api/internal/outreach/clients"

	var out SuccessResponse[RegisterClientResponse]
	if err := c.NewRequest(ctx, http.MethodPost, path, req, &out).WithApiKey(c.apiKey).doJSON(); err != nil {
		return nil, err
	}

	return &out.Data, nil
}

// Subscribe subscribes a client to an outreach job (requires outreach client key)
func (c *Client) Subscribe(ctx context.Context, clientKey string, req *SubscribeRequest) (*SubscribeResponse, error) {
	path := "/api/internal/outreach/subscriptions"

	var out SuccessResponse[SubscribeResponse]
	if err := c.NewRequest(ctx, http.MethodPost, path, req, &out).WithOutreachClientKey(clientKey).doJSON(); err != nil {
		return nil, err
	}

	return &out.Data, nil
}

// Unsubscribe removes a client's subscription from an outreach job (requires outreach client key)
func (c *Client) Unsubscribe(ctx context.Context, clientKey string, req *UnsubscribeRequest) error {
	path := "/api/internal/outreach/subscriptions"

	return c.NewRequest(ctx, http.MethodDelete, path, req, nil).WithOutreachClientKey(clientKey).doJSON()
}
