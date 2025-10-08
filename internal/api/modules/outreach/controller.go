package outreach_module

import (
	"net/http"

	"github.com/ethanbaker/assistant/pkg/sdk"
	"github.com/gin-gonic/gin"
)

// RegisterImplementation handles POST requests to register a new implementation
func RegisterImplementation(c *gin.Context) {
	// Parse request body
	var req sdk.OutreachRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(sdk.NewErrorResponse(http.StatusBadRequest, "Could not parse request body", err).AsGinResponse())
		return
	}

	// Get service and register implementation
	if err := outreachService.RegisterImplementation(&req); err != nil {
		c.JSON(sdk.NewErrorResponse(http.StatusInternalServerError, "Failed to register implementation", err).AsGinResponse())
		return
	}

	// Return success response
	resp := &sdk.OutreachRegisterResponse{ClientId: req.ClientId}
	c.JSON(sdk.NewSuccessResponse("Implementation registered successfully", resp).AsGinResponse())
}

// UnregisterImplementation handles DELETE requests to unregister an implementation
func UnregisterImplementation(c *gin.Context) {
	// Get client ID from body
	var req sdk.OutreachUnregisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(sdk.NewErrorResponse(http.StatusBadRequest, "Could not parse request body", err).AsGinResponse())
		return
	}

	// Get service and unregister implementation
	if err := outreachService.UnregisterImplementation(req.ClientId); err != nil {
		c.JSON(sdk.NewErrorResponse(http.StatusInternalServerError, "Failed to unregister implementation", err).AsGinResponse())
		return
	}

	c.JSON(sdk.NewSuccess("Implementation unregistered successfully").AsGinResponse())
}

// GetImplementations handles GET requests to list all registered implementations
func GetImplementations(c *gin.Context) {
	// Get service and list implementations
	implementations, err := outreachService.GetImplementations()
	if err != nil {
		c.JSON(sdk.NewErrorResponse(http.StatusInternalServerError, "Failed to get implementations", err).AsGinResponse())
		return
	}

	c.JSON(sdk.NewSuccessResponse("Implementations retrieved successfully", implementations).AsGinResponse())
}

// GetStatus handles GET requests to get the current status of the outreach service
func GetStatus(c *gin.Context) {
	// Get service and status
	status := outreachService.GetStatus()

	c.JSON(sdk.NewSuccessResponse("Status retrieved successfully", status).AsGinResponse())
}
