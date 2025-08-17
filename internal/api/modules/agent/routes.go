package agent

import (
	"github.com/ethanbaker/api/pkg/api_key"
	"github.com/gin-gonic/gin"
)

// Register routes for the agent module
func RegisterRoutes(g *gin.RouterGroup) {
	// Create base group for agent routes
	group := g.Group("/agent")
	group.Handlers = append(group.Handlers, api_key.APIKeyHeaderHandler(validateAPIKey))

	// Session management routes
	group.POST("/sessions", CreateSession)             // Create a new session
	group.GET("/sessions/:uuid", GetSession)           // Get an existing session by UUID
	group.POST("/sessions/:uuid/message", PostMessage) // Add a message to an existing session
	group.DELETE("/sessions/:uuid", DeleteSession)     // Delete an existing session
}
