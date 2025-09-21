package outreach_module

import (
	"github.com/gin-gonic/gin"
)

// Register routes for the outreach module
func RegisterRoutes(g *gin.RouterGroup) {
	// Create base group for outreach routes
	group := g.Group("/outreach")

	// Public routes (registration)
	group.POST("/implementations", RegisterImplementation)

	// Protected routes (require authentication)
	protected := group.Group("/")
	protected.Use(AuthenticationHandler())
	protected.DELETE("/implementations", UnregisterImplementation)
	protected.GET("/implementations", GetImplementations)
	protected.GET("/status", GetStatus)
}
