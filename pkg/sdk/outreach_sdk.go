package sdk

import (
	"context"
	"net/http"
)

// RegisterImplementation registers a new outreach implementation
func (c *Client) RegisterImplementation(ctx context.Context, req *OutreachRegisterRequest) (*OutreachRegisterResponse, error) {
	path := "/api/outreach/implementations"

	var out SuccessResponse[OutreachRegisterResponse]
	if err := c.NewRequest(ctx, http.MethodPost, path, req, &out).WithApiKey(c.apiKey).doJSON(); err != nil {
		return nil, err
	}

	return &out.Data, nil
}

// UnregisterImplementation removes an outreach implementation
func (c *Client) UnregisterImplementation(ctx context.Context, clientId string, creds OutreachCredentials) error {
	path := "/api/outreach/implementations/"
	req := &OutreachUnregisterRequest{ClientId: clientId}

	var out SuccessResponse[map[string]string]
	if err := c.NewRequest(ctx, http.MethodDelete, path, req, &out).WithClientCredentials(creds.ClientId, creds.ClientSecret).doJSON(); err != nil {
		return err
	}

	return nil
}

// GetImplementations retrieves all registered implementations
func (c *Client) GetImplementations(ctx context.Context, creds OutreachCredentials) (*OutreachListImplementationsResponse, error) {
	path := "/api/outreach/implementations"

	var out SuccessResponse[OutreachListImplementationsResponse]
	if err := c.NewRequest(ctx, http.MethodGet, path, nil, &out).WithClientCredentials(creds.ClientId, creds.ClientSecret).doJSON(); err != nil {
		return nil, err
	}

	return &out.Data, nil
}

// GetOutreachStatus retrieves the current status of the outreach service
func (c *Client) GetOutreachStatus(ctx context.Context, creds OutreachCredentials) (*OutreachStatusResponse, error) {
	path := "/api/outreach/status"

	var out SuccessResponse[OutreachStatusResponse]
	if err := c.NewRequest(ctx, http.MethodGet, path, nil, &out).WithClientCredentials(creds.ClientId, creds.ClientSecret).doJSON(); err != nil {
		return nil, err
	}

	return &out.Data, nil
}
