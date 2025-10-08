package agent

import (
	"fmt"
	"log"

	"github.com/ethanbaker/api/pkg/api_key"
	"github.com/ethanbaker/assistant/pkg/utils"
	"github.com/gin-gonic/gin"
)

// Register routes for the agent module
func RegisterRoutes(g *gin.RouterGroup, cfg *utils.Config) {
	// Make api key validator
	validator, err := makeApiKeyValidator(cfg)
	if err != nil {
		log.Fatalf("failed to create API key validator: %v", err)
	}

	// Create base group for agent routes
	group := g.Group("/agent")
	group.Handlers = append(group.Handlers, api_key.APIKeyHeaderHandler(validator))

	// Session management routes
	group.POST("/sessions", CreateSession)             // Create a new session
	group.GET("/sessions/:uuid", GetSession)           // Get an existing session by UUID
	group.POST("/sessions/:uuid/message", PostMessage) // Add a message to an existing session
	group.DELETE("/sessions/:uuid", DeleteSession)     // Delete an existing session
}

// makeApiKeyValidator checks if the provided API key is valid
func makeApiKeyValidator(cfg *utils.Config) (func(key string) bool, error) {
	// Get api key from config
	apiKey := cfg.Get("API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("API_KEY not set in environment")
	}

	return func(key string) bool {
		return apiKey == key
	}, nil
}
