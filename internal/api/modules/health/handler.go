package health

import (
	"github.com/ethanbaker/assistant/internal/api/routes"
	"github.com/ethanbaker/assistant/internal/logger"
	"github.com/ethanbaker/assistant/pkg/sdk"
	"github.com/gin-gonic/gin"
)

// Handler defines dependencies for the health handler
type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

// Return status of the API
func (h *Handler) GetStatus(c *gin.Context) {
	logger.Debugf("GET '%s' endpoint hit", routes.GET_HEALTH)

	res := sdk.NewSuccessMessage("OK")
	c.JSON(res.AsGinResponse())
}
