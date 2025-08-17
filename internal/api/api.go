package api

import (
	"log"
	"strings"
	"time"

	api_utils "github.com/ethanbaker/api/pkg/utils"
	"github.com/ethanbaker/assistant/pkg/utils"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	agent_module "github.com/ethanbaker/assistant/internal/api/modules/agent"
	health_module "github.com/ethanbaker/assistant/internal/api/modules/health"
)

func Start(cfg *utils.Config) {
	// Initialized configuration settings
	port := cfg.GetWithDefault("API_PORT", "8080")

	// Add app level settings/routes
	engine := gin.Default()
	engine.NoRoute(api_utils.NoRouteHandler)

	// Add trusted proxies
	engine.SetTrustedProxies(nil)

	// Add CORS using gin-contrib/cors (https://github.com/gin-contrib/cors for documentation)
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Split(cfg.GetWithDefault("CORS_ALLOWED_ORIGINS", "*"), ","),
		AllowMethods:     []string{"OPTIONS", "GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	// Base group '/api' for all API routes
	baseGroup := engine.Group("/api")

	// Adding custom modules
	health_module.RegisterRoutes(baseGroup)

	agent_module.RegisterRoutes(baseGroup)
	agent_module.Init(cfg)

	// Then after performing initial setup, start the server
	if err := engine.Run(":" + port); err != nil {
		log.Fatal("[API-MAIN]: Failed to start server: ", err)
	}
}
