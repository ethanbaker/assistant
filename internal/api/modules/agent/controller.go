package agent

import (
	"fmt"
	"net/http"

	"github.com/ethanbaker/assistant/internal/stores/session"
	"github.com/ethanbaker/assistant/pkg/sdk"
	"github.com/gin-gonic/gin"
)

// CreateSession handles POST requests to create a new session
func CreateSession(c *gin.Context) {
	// Parse request body
	var req sdk.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(sdk.NewErrorResponse(http.StatusBadRequest, "Could not parse request body", err).AsGinResponse())
		return
	}

	// Create a new session using the orchestrator
	orchestrator := GetOrchestrator()
	session, err := orchestrator.NewSession(c.Request.Context(), req.UserID)
	if err != nil {
		c.JSON(sdk.NewErrorResponse(http.StatusInternalServerError, "Failed to create session", err).AsGinResponse())
		return
	}

	c.JSON(sdk.NewSuccessResponse("Session created successfully", toSDKSession(session)).AsGinResponse())
}

// GetSession handles GET requests to retrieve an existing session by UUID
func GetSession(c *gin.Context) {
	uuid := c.Param("uuid")

	// Retrieve the session using the orchestrator
	orchestrator := GetOrchestrator()
	session, err := orchestrator.FindSession(c.Request.Context(), uuid)
	if err != nil {
		c.JSON(sdk.NewErrorResponse(http.StatusBadRequest, "Session not found", err).AsGinResponse())
		return
	}

	c.JSON(sdk.NewSuccessResponse("Session retrieved successfully", toSDKSession(session)).AsGinResponse())
}

// PostMessage handles POST requests to add a message to an existing session
func PostMessage(c *gin.Context) {
	uuid := c.Param("uuid")

	// Parse request body
	var req sdk.PostMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(sdk.NewErrorResponse(http.StatusBadRequest, "Could not parse request body", err).AsGinResponse())
		return
	}

	orchestrator := GetOrchestrator()

	// Record current item count
	count, err := orchestrator.GetItemCountBySessionID(c.Request.Context(), uuid)
	if err != nil {
		c.JSON(sdk.NewErrorResponse(http.StatusInternalServerError, "Failed to get item count", err).AsGinResponse())
		return
	}

	// Add the message to the session using the orchestrator
	msg, err := orchestrator.AddMessage(c.Request.Context(), uuid, req)
	if err != nil {
		c.JSON(sdk.NewErrorResponse(http.StatusInternalServerError, "Failed to add message", err).AsGinResponse())
		return
	}

	// Get new item count
	newCount, err := orchestrator.GetItemCountBySessionID(c.Request.Context(), uuid)
	if err != nil {
		c.JSON(sdk.NewErrorResponse(http.StatusInternalServerError, "Failed to get new item count", err).AsGinResponse())
		return
	}
	newCount-- // Adjust for user message

	// Handle case where no new item was added
	if newCount == count {
		c.JSON(sdk.NewErrorResponse(http.StatusInternalServerError, "Agent returned no response", nil).AsGinResponse())
		return
	}

	// Items are stored in mysql, so fetch them to get the full data before returning
	items, err := orchestrator.GetDBItemsBySessionID(c.Request.Context(), uuid, newCount-count)
	if err != nil {
		c.JSON(sdk.NewErrorResponse(http.StatusInternalServerError, "Failed to get added items", err).AsGinResponse())
		return
	}

	var dbItems []sdk.Item
	for _, item := range items {
		dbItems = append(dbItems, toSDKItem(item))
	}

	// Construct response
	resp := sdk.PostMessageResponse{
		FinalOutput: fmt.Sprint(msg.FinalOutput),
		Items:       dbItems,
	}

	c.JSON(sdk.NewSuccessResponse("Message sent successfully", resp).AsGinResponse())
}

// DeleteSession handles DELETE requests to remove an existing session
func DeleteSession(c *gin.Context) {
	uuid := c.Param("uuid")

	// Delete the session using the orchestrator
	orchestrator := GetOrchestrator()
	sess, err := orchestrator.RemoveSession(c.Request.Context(), uuid)
	if err != nil {
		c.JSON(sdk.NewErrorResponse(http.StatusInternalServerError, "Failed to delete session", err).AsGinResponse())
		return
	}

	c.JSON(sdk.NewSuccessResponse("Session deleted successfully", sess).AsGinResponse())
}

// Helper method to reverse slices
func reverse[T any](s []T) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// Helper method to convert internal session to sdk session
func toSDKSession(session *session.Session) sdk.Session {
	resp := sdk.Session{
		ID:        session.ID.String(),
		CreatedAt: session.CreatedAt,
		UpdatedAt: session.UpdatedAt,
		DeletedAt: session.DeletedAt,
		UserID:    session.UserID,
	}

	for _, item := range session.Items {
		dbItem := toSDKItem(*item)
		resp.Items = append(resp.Items, &dbItem)
	}

	return resp
}

// Helper method to convert internal item to sdk item
func toSDKItem(item session.Item) sdk.Item {
	return sdk.Item{
		ID:        item.ID,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
		DeletedAt: item.DeletedAt,
		SessionID: item.SessionID,
		Data:      sdk.ResponseItemData(item.ResponseItem),
	}
}
