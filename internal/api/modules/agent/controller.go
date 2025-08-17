package agent

import (
	"net/http"

	"github.com/ethanbaker/api/pkg/api_types"
	"github.com/ethanbaker/assistant/pkg/sdk"
	"github.com/gin-gonic/gin"
)

// CreateSession handles POST requests to create a new session
func CreateSession(c *gin.Context) {
	// Parse request body
	var req sdk.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(api_types.NewErrorResponse(http.StatusBadRequest, err.Error()).AsGinResponse())
		return
	}

	// Create a new session using the orchestrator
	orchestrator := GetOrchestrator()
	session, err := orchestrator.NewSession(c.Request.Context(), req.UserID)
	if err != nil {
		c.JSON(api_types.NewErrorResponse(http.StatusInternalServerError, "Failed to create session").AsGinResponse())
		return
	}

	c.JSON(api_types.NewSuccessResponse("Session created successfully", session).AsGinResponse())
}

// GetSession handles GET requests to retrieve an existing session by UUID
func GetSession(c *gin.Context) {
	uuid := c.Param("uuid")

	// Retrieve the session using the orchestrator
	orchestrator := GetOrchestrator()
	session, err := orchestrator.FindSession(c.Request.Context(), uuid)
	if err != nil {
		c.JSON(api_types.NewErrorResponse(http.StatusBadRequest, "Session not found").AsGinResponse())
		return
	}

	c.JSON(api_types.NewSuccessResponse("Session retrieved successfully", session).AsGinResponse())
}

// PostMessage handles POST requests to add a message to an existing session
func PostMessage(c *gin.Context) {
	uuid := c.Param("uuid")

	// Parse request body
	var req sdk.PostMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(api_types.NewErrorResponse(http.StatusBadRequest, err.Error()).AsGinResponse())
		return
	}

	// Add the message to the session using the orchestrator
	orchestrator := GetOrchestrator()
	msg, err := orchestrator.AddMessage(c.Request.Context(), uuid, req)
	if err != nil {
		c.JSON(api_types.NewErrorResponse(http.StatusInternalServerError, err.Error()).AsGinResponse())
		return
	}

	// Construct response
	resp := sdk.PostMessageResponse{
		Output: msg.FinalOutput,
	}

	c.JSON(api_types.NewSuccessResponse("Message added successfully", resp).AsGinResponse())
}

// DeleteSession handles DELETE requests to remove an existing session
func DeleteSession(c *gin.Context) {
	uuid := c.Param("uuid")

	// Delete the session using the orchestrator
	orchestrator := GetOrchestrator()
	sess, err := orchestrator.RemoveSession(c.Request.Context(), uuid)
	if err != nil {
		c.JSON(api_types.NewErrorResponse(http.StatusInternalServerError, "Failed to delete session").AsGinResponse())
		return
	}

	c.JSON(api_types.NewSuccessResponse("Session deleted successfully", sess).AsGinResponse())
}
