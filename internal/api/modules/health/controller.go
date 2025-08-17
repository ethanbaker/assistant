package health

import (
	"github.com/ethanbaker/api/pkg/api_types"
	"github.com/gin-gonic/gin"
)

// Return status of the API
func getStatus(c *gin.Context) {
	res := api_types.NewSuccessResponse("OK", nil)
	c.JSON(res.AsGinResponse())
}
