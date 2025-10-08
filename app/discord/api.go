package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	api_utils "github.com/ethanbaker/api/pkg/utils"
	"github.com/ethanbaker/assistant/pkg/sdk"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const API_PREFIX = "/api"
const OUTREACH_ENDPOINT = "/outreach-message"

// registerOutreach registers the API with the outreach module
func (b *Bot) registerOutreach() error {
	// Get credentials from config
	clientID := b.config.Get("OUTREACH_CLIENT_ID")
	if clientID == "" {
		return fmt.Errorf("OUTREACH_CLIENT_ID not set in environment")
	}

	clientSecret := b.config.Get("OUTREACH_CLIENT_SECRET")
	if clientSecret == "" {
		return fmt.Errorf("OUTREACH_CLIENT_SECRET not set in environment")
	}

	host := b.config.Get("OUTREACH_HOST")
	if host == "" {
		return fmt.Errorf("OUTREACH_HOST not set in environment")
	}

	_, err := b.api.RegisterImplementation(context.Background(), &sdk.OutreachRegisterRequest{
		ClientId:     clientID,
		ClientSecret: clientSecret,
		CallbackUrl:  fmt.Sprintf("%s%s%s", host, API_PREFIX, OUTREACH_ENDPOINT),
	})

	if err != nil {
		return fmt.Errorf("failed to register outreach implementation: %v", err)
	}
	return nil
}

// unregisterOutreach unregisters the API from the outreach module
func (b *Bot) unregisterOutreach() error {
	clientID := b.config.Get("OUTREACH_CLIENT_ID")
	if clientID == "" {
		return fmt.Errorf("OUTREACH_CLIENT_ID not set in environment")
	}

	clientSecret := b.config.Get("OUTREACH_CLIENT_SECRET")
	if clientSecret == "" {
		return fmt.Errorf("OUTREACH_CLIENT_SECRET not set in environment")
	}

	return b.api.UnregisterImplementation(context.Background(), clientID, sdk.OutreachCredentials{
		ClientId:     clientID,
		ClientSecret: clientSecret,
	})
}

// startAPI starts the Discord API server
func (b *Bot) startAPI() {
	cfg := b.config

	// Initialized configuration settings
	port := cfg.GetWithDefault("API_PORT", "8081") // Different port from main API

	// Add app level settings/routes
	engine := gin.Default()
	engine.NoRoute(api_utils.NoRouteHandler)

	// Add trusted proxies
	engine.SetTrustedProxies(nil)

	// Add CORS using gin-contrib/cors
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Split(cfg.GetWithDefault("CORS_ALLOWED_ORIGINS", "*"), ","),
		AllowMethods:     []string{"OPTIONS", "GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	// Base group '/api' for all API routes
	baseGroup := engine.Group(API_PREFIX)
	baseGroup.POST(OUTREACH_ENDPOINT, b.OnOutreachMessage)

	// Start the server
	log.Printf("[DISCORD-API]: Starting Discord API server on port %s", port)
	if err := engine.Run(":" + port); err != nil {
		log.Fatal("[DISCORD-API]: Failed to start server: ", err)
	}
}

// idempotencyStore is a map to track idempotency keys
var idempotencyStore = make(map[string]bool)

// OnOutreachMessage handles POST requests to send outreach messages via Discord DM
func (b *Bot) OnOutreachMessage(c *gin.Context) {
	// Parse request body
	var req sdk.OutreachRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(sdk.NewErrorResponse(http.StatusBadRequest, "Could not parse request body", err).AsGinResponse())
		return
	}

	// Check for idempotency
	if req.Id == "" {
		c.JSON(sdk.NewErrorResponse(http.StatusBadRequest, "Id is required", nil).AsGinResponse())
		return
	}
	if complete, ok := idempotencyStore[req.Id]; ok && complete {
		c.JSON(sdk.NewSuccess("Outreach message already sent").AsGinResponse())
		return
	}
	idempotencyStore[req.Id] = false

	// Get the user ID from bot
	userID := b.userID
	if userID == "" {
		c.JSON(sdk.NewErrorResponse(http.StatusInternalServerError, "USER_ID not configured in environment", nil).AsGinResponse())
		return
	}

	// Validate that we have a Discord session
	if b.dg == nil {
		c.JSON(sdk.NewErrorResponse(http.StatusInternalServerError, "Discord session not initialized", nil).AsGinResponse())
		return
	}

	// Create a direct message channel with the user
	dmChannel, err := b.dg.UserChannelCreate(userID)
	if err != nil {
		log.Printf("[DISCORD-OUTREACH]: Failed to create DM channel with user %s: %v", userID, err)
		c.JSON(sdk.NewErrorResponse(http.StatusInternalServerError, "Failed to create DM channel", err).AsGinResponse())
		return
	}

	// Send the message content to the user
	replySanitizeHTML(b.dg, dmChannel.ID, req.Content)
	idempotencyStore[req.Id] = true

	// Log successful message send
	log.Printf("[DISCORD-OUTREACH]: Successfully sent outreach message to user %s (ID: %s, Key: %s)", userID, req.Id, req.Key)

	// Return success response
	c.JSON(sdk.NewSuccess("Outreach message sent successfully").AsGinResponse())
}
