package outreach_module

import (
	"fmt"
	"log"

	"github.com/ethanbaker/api/pkg/api_key"
	"github.com/ethanbaker/assistant/pkg/utils"
	"github.com/gin-gonic/gin"
)

// Register routes for the outreach module
func RegisterRoutes(g *gin.RouterGroup, cfg *utils.Config) {
	// Make api key validator
	validator, err := makeApiKeyValidator(cfg)
	if err != nil {
		log.Fatalf("failed to create API key validator: %v", err)
	}

	// Create base group for outreach routes
	group := g.Group("/outreach")

	// Public routes (require api key)
	group.POST("/implementations", RegisterImplementation, api_key.APIKeyHeaderHandler(validator))

	// Protected routes (require authentication)
	protected := group.Group("/")
	protected.Use(AuthenticationHandler())
	protected.DELETE("/implementations", UnregisterImplementation)
	protected.GET("/implementations", GetImplementations)
	protected.GET("/status", GetStatus)
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
