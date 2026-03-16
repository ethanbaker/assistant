package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ethanbaker/assistant/internal/domain"
	clientrepo "github.com/ethanbaker/assistant/internal/repositories/client"
	"github.com/ethanbaker/assistant/internal/repositories/session"
	"github.com/ethanbaker/assistant/pkg/sdk"
	"github.com/gin-gonic/gin"
	"github.com/nlpodyssey/openai-agents-go/memory"
)

// HandlerConfig defines configuration for the handler
type HandlerConfig struct {
	Service          *Service
	ClientRepository clientrepo.Repository
}

// Handler defines dependencies for the agent handler
type Handler struct {
	HandlerConfig
}

// NewHandler creates a new agent handler instance
func NewHandler(cfg HandlerConfig) *Handler {
	return &Handler{
		HandlerConfig: cfg,
	}
}

// CreateSession handles POST requests to create a new session
func (h *Handler) CreateSession(c *gin.Context) {
	// Parse request body
	var req sdk.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(sdk.NewBadRequest("Could not parse request body").WithDetails(err.Error()).AsGinResponse())
		return
	}

	// Create a new session using the service
	session, err := h.Service.CreateSession(c.Request.Context(), req.UserID)
	if err != nil {
		c.JSON(sdk.NewInternalServerError("Failed to create session").WithDetails(err.Error()).AsGinResponse())
		return
	}

	c.JSON(sdk.NewSuccess(toSDKSession(session)).AsGinResponse())
}

// GetSession handles GET requests to retrieve an existing session by UUID
func (h *Handler) GetSession(c *gin.Context) {
	uuid := c.Param("uuid")

	// Retrieve the session using the service
	session, err := h.Service.GetSession(c.Request.Context(), uuid)
	if err != nil {
		c.JSON(sdk.NewBadRequest("Session not found").WithDetails(err.Error()).AsGinResponse())
		return
	}

	c.JSON(sdk.NewSuccess(toSDKSession(session)).AsGinResponse())
}

// PostMessage handles POST requests to add a message to an existing session
func (h *Handler) PostMessage(c *gin.Context) {
	uuid := c.Param("uuid")

	// Parse request body
	var req sdk.PostMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(sdk.NewBadRequest("Could not parse request body").WithDetails(err.Error()).AsGinResponse())
		return
	}

	// Validate session exists
	_, err := h.Service.GetSession(c.Request.Context(), uuid)
	if err != nil {
		c.JSON(sdk.NewBadRequest("Session not found").WithDetails(err.Error()).AsGinResponse())
		return
	}

	// Record current item count
	count, err := h.Service.GetItemCount(c.Request.Context(), uuid)
	if err != nil {
		c.JSON(sdk.NewInternalServerError("Failed to get item count").WithDetails(err.Error()).AsGinResponse())
		return
	}

	// Add the message to the session using the service
	msg, err := h.Service.AddMessage(c.Request.Context(), uuid, req)
	if err != nil {
		c.JSON(sdk.NewInternalServerError("Failed to add message").WithDetails(err.Error()).AsGinResponse())
		return
	}

	// Get new item count
	newCount, err := h.Service.GetItemCount(c.Request.Context(), uuid)
	if err != nil {
		c.JSON(sdk.NewInternalServerError("Failed to get new item count").WithDetails(err.Error()).AsGinResponse())
		return
	}
	newCount-- // Adjust for user message

	// Handle case where no new item was added
	if newCount == count {
		c.JSON(sdk.NewInternalServerError("Agent returned no response").AsGinResponse())
		return
	}

	// Items are stored in mysql, so fetch them to get the full data before returning
	items, err := h.Service.GetSessionItems(c.Request.Context(), uuid, newCount-count)
	if err != nil {
		c.JSON(sdk.NewInternalServerError("Failed to get added items").WithDetails(err.Error()).AsGinResponse())
		return
	}

	var dbItems []sdk.Item
	for _, item := range items {
		dbItems = append(dbItems, toSDKItem(*item))
	}

	// Construct response
	resp := sdk.PostMessageResponse{
		FinalOutput: fmt.Sprint(msg.FinalOutput),
		Items:       dbItems,
	}

	c.JSON(sdk.NewSuccess(resp).AsGinResponse())
}

// DeleteSession handles DELETE requests to remove an existing session
func (h *Handler) DeleteSession(c *gin.Context) {
	uuid := c.Param("uuid")

	// Make sure the session exists
	_, err := h.Service.GetSession(c.Request.Context(), uuid)
	if err != nil {
		c.JSON(sdk.NewBadRequest("Session not found").WithDetails(err.Error()).AsGinResponse())
		return
	}

	// Delete the session using the service
	sess, err := h.Service.DeleteSession(c.Request.Context(), uuid)
	if err != nil {
		c.JSON(sdk.NewInternalServerError("Failed to delete session").WithDetails(err.Error()).AsGinResponse())
		return
	}

	c.JSON(sdk.NewSuccess(toSDKSession(sess)).AsGinResponse())
}

// AttachJobExecutionContext handles POST requests to add outreach execution context to a session.
func (h *Handler) AttachJobExecutionContext(c *gin.Context) {
	uuid := c.Param("uuid")

	// Find client from api key in header
	apiKey := strings.TrimSpace(c.GetHeader("X-Client-Key"))
	if apiKey == "" {
		c.JSON(sdk.NewUnauthorized("missing client key").AsGinResponse())
		return
	}

	if h.ClientRepository == nil {
		c.JSON(sdk.NewInternalServerError("Client repository is not configured").AsGinResponse())
		return
	}

	client, err := h.ClientRepository.FindByApiKey(apiKey)
	if err != nil {
		c.JSON(sdk.NewInternalServerError("Failed to authenticate client").WithDetails(err.Error()).AsGinResponse())
		return
	}
	if client == nil {
		c.JSON(sdk.NewUnauthorized("invalid client key").AsGinResponse())
		return
	}

	// Parse parameters
	var req AttachJobExecutionContextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(sdk.NewBadRequest("Could not parse request body").WithDetails(err.Error()).AsGinResponse())
		return
	}

	// Attach job context
	err = h.Service.AttachJobExecutionsContext(c.Request.Context(), uuid, int(client.ID), req.JobExecutionIDs)
	if err != nil {
		switch {
		case errors.Is(err, ErrExecutionNotFound):
			c.JSON(sdk.NewBadRequest(err.Error()).AsGinResponse())
		case errors.Is(err, ErrExecutionForbidden):
			c.JSON(sdk.NewForbidden().WithDetails(err.Error()).AsGinResponse())
		default:
			c.JSON(sdk.NewInternalServerError("Failed to attach job execution context").WithDetails(err.Error()).AsGinResponse())
		}
		return
	}

	c.JSON(sdk.NewSuccessMessage("Job execution context attached").AsGinResponse())
}

// Helper method to convert internal session to sdk session
func toSDKSession(s memory.Session) sdk.Session {
	// Cast to concrete type to access fields
	switch s := s.(type) {
	case *session.SessionEntity:
		resp := sdk.Session{
			ID:        s.SessionID(context.Background()),
			CreatedAt: s.CreatedAt,
			UpdatedAt: s.UpdatedAt,
			DeletedAt: s.DeletedAt,
			UserID:    s.UserID,
		}

		for _, item := range s.Items {
			dbItem := toSDKItem(*item)
			resp.Items = append(resp.Items, &dbItem)
		}
		return resp
	}

	// Unhandled type
	return sdk.Session{}
}

// Helper method to convert internal item to sdk item
func toSDKItem(item domain.Item) sdk.Item {
	return sdk.Item{
		ID:        item.ID,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
		DeletedAt: item.DeletedAt,
		SessionID: item.SessionID,
		Data:      sdk.ResponseItemData{TResponseInputItem: item.ResponseItem.TResponseInputItem},
	}
}
