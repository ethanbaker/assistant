package sdk

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ethanbaker/api/pkg/api_types"
)

// RegisterImplementation registers a new outreach implementation
func (c *Client) RegisterImplementation(ctx context.Context, req *OutreachRegisterRequest) (*OutreachRegisterResponse, error) {
	path := "/api/outreach/implementations"

	var out ApiResponse[OutreachRegisterResponse]
	if err := c.NewRequest(ctx, http.MethodPost, path, req, &out).WithApiKey(c.apiKey).doJSON(); err != nil {
		return nil, err
	}

	// Check for success
	switch out.Status {
	case api_types.StatusFail:
		return nil, fmt.Errorf("failed to register implementation: %s", out.Message)
	case api_types.StatusError:
		return nil, fmt.Errorf("error registering implementation (%s): %v", out.Message, out.Error)
	}

	return &out.Data, nil
}

// UnregisterImplementation removes an outreach implementation
func (c *Client) UnregisterImplementation(ctx context.Context, clientId string, creds OutreachCredentials) error {
	path := "/api/outreach/implementations/"
	req := &OutreachUnregisterRequest{ClientId: clientId}

	var out ApiResponse[map[string]string]
	if err := c.NewRequest(ctx, http.MethodDelete, path, req, &out).WithClientCredentials(creds.ClientId, creds.ClientSecret).doJSON(); err != nil {
		return err
	}

	// Check for success
	switch out.Status {
	case api_types.StatusFail:
		return fmt.Errorf("failed to unregister implementation: %s", out.Message)
	case api_types.StatusError:
		return fmt.Errorf("error unregistering implementation (%s): %v", out.Message, out.Error)
	}

	return nil
}

// GetImplementations retrieves all registered implementations
func (c *Client) GetImplementations(ctx context.Context, creds OutreachCredentials) (*OutreachListImplementationsResponse, error) {
	path := "/api/outreach/implementations"

	var out ApiResponse[OutreachListImplementationsResponse]
	if err := c.NewRequest(ctx, http.MethodGet, path, nil, &out).WithClientCredentials(creds.ClientId, creds.ClientSecret).doJSON(); err != nil {
		return nil, err
	}

	// Check for success
	switch out.Status {
	case api_types.StatusFail:
		return nil, fmt.Errorf("failed to get implementations: %s", out.Message)
	case api_types.StatusError:
		return nil, fmt.Errorf("error getting implementations (%s): %v", out.Message, out.Error)
	}

	return &out.Data, nil
}

// GetOutreachStatus retrieves the current status of the outreach service
func (c *Client) GetOutreachStatus(ctx context.Context, creds OutreachCredentials) (*OutreachStatusResponse, error) {
	path := "/api/outreach/status"

	var out ApiResponse[OutreachStatusResponse]
	if err := c.NewRequest(ctx, http.MethodGet, path, nil, &out).WithClientCredentials(creds.ClientId, creds.ClientSecret).doJSON(); err != nil {
		return nil, err
	}

	// Check for success
	switch out.Status {
	case api_types.StatusFail:
		return nil, fmt.Errorf("failed to get outreach status: %s", out.Message)
	case api_types.StatusError:
		return nil, fmt.Errorf("error getting outreach status (%s): %v", out.Message, out.Error)
	}

	return &out.Data, nil
}
