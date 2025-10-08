package outreach_module

import (
	"net/http"
	"strings"

	"github.com/ethanbaker/assistant/pkg/sdk"
	"github.com/gin-gonic/gin"
)

// AuthenticationHandler middleware validates client credentials for protected endpoints
// The credentials can be provided via Basic Auth or Bearer token in the Authorization header
// For basic, use standard Basic auth header with client ID as username and client secret as password
// For bearer, use "Bearer <token>" format where <token> is "clientid:clientsecret"
func AuthenticationHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get credentials from Authorization header (Basic auth format)
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(sdk.NewErrorResponse(http.StatusUnauthorized, "Authorization header required", nil).AsGinResponse())
			c.Abort()
			return
		}

		// Check for Bearer token format or Basic auth
		var clientID, clientSecret string

		if strings.HasPrefix(authHeader, "Bearer ") {
			// Extract client ID and secret from Bearer token
			// Format: "Bearer clientid:clientsecret"
			token := strings.TrimPrefix(authHeader, "Bearer ")
			parts := strings.SplitN(token, ":", 2)
			if len(parts) != 2 {
				c.JSON(sdk.NewErrorResponse(http.StatusUnauthorized, "Invalid authorization format. Use Bearer clientid:clientsecret", nil).AsGinResponse())
				c.Abort()
				return
			}
			clientID = parts[0]
			clientSecret = parts[1]
		} else if strings.HasPrefix(authHeader, "Basic ") {
			// Extract from Basic auth
			username, password, ok := c.Request.BasicAuth()
			if !ok {
				c.JSON(sdk.NewErrorResponse(http.StatusUnauthorized, "Invalid basic authentication", nil).AsGinResponse())
				c.Abort()
				return
			}
			clientID = username
			clientSecret = password
		} else {
			c.JSON(sdk.NewErrorResponse(http.StatusUnauthorized, "Invalid authorization format. Use Bearer or Basic authentication", nil).AsGinResponse())
			c.Abort()
			return
		}

		// Validate credentials using the outreach service
		impl, err := outreachService.AuthenticateClient(clientID, clientSecret)
		if err != nil {
			c.JSON(sdk.NewErrorResponse(http.StatusUnauthorized, "Invalid client credentials", err).AsGinResponse())
			c.Abort()
			return
		}

		// Store implementation in context for use by handlers
		c.Set("authenticated_client", impl)
		c.Set("client_id", clientID)

		c.Next()
	}
}

// GetAuthenticatedClient retrieves the authenticated client from the gin context
func GetAuthenticatedClient(c *gin.Context) (string, bool) {
	clientID, exists := c.Get("client_id")
	if !exists {
		return "", false
	}

	if id, ok := clientID.(string); ok {
		return id, true
	}

	return "", false
}
