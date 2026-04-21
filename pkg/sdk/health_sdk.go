package sdk

import (
	"context"
	"net/http"
)

// GetHealth checks the health status of the API
func (c *Client) GetHealth(ctx context.Context) error {
	path := "/api/public/health"

	return c.NewRequest(ctx, http.MethodGet, path, nil, nil).doJSON()
}
