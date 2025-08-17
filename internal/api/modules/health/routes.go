package health

import "github.com/gin-gonic/gin"

// RegisterRoutes registers the routes for the health module
func RegisterRoutes(g *gin.RouterGroup) {
	g.GET("/health", getStatus)
}
