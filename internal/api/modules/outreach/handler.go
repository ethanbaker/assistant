package outreach

import (
	"net/http"
	"strings"

	"github.com/ethanbaker/assistant/internal/domain"
	"github.com/ethanbaker/assistant/pkg/sdk"
	"github.com/gin-gonic/gin"
)

// HandlerConfig contains outreach handler dependencies.
type HandlerConfig struct {
	AdminKey            string
	JobService          *JobService
	ClientService       *ClientService
	SubscriptionService *SubscriptionService
}

// Handler serves outreach API endpoints.
type Handler struct {
	HandlerConfig
}

func NewHandler(cfg HandlerConfig) *Handler {
	return &Handler{HandlerConfig: cfg}
}

// CreateJob handles POST /outreach/jobs.
func (h *Handler) CreateJob(c *gin.Context) {
	if !h.authorizeAdmin(c.GetHeader("X-Admin-Key")) {
		c.JSON(sdk.NewUnauthorized("invalid api key").AsGinResponse())
		return
	}

	var req CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(sdk.NewBadRequest("Could not parse request body").WithDetails(err.Error()).AsGinResponse())
		return
	}

	job := &domain.Job{
		Name:       req.Name,
		Active:     true,
		Schedule:   req.Schedule,
		Handler:    req.Handler,
		Parameters: req.Parameters,
	}
	if req.Active != nil {
		job.Active = *req.Active
	}

	if err := h.JobService.CreateJob(job); err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(sdk.NewSuccess(CreateJobResponse{JobID: int(job.ID)}).AsGinResponse())
}

// RegisterClient handles POST /outreach/clients.
func (h *Handler) RegisterClient(c *gin.Context) {
	if !h.authorizeAdmin(c.GetHeader("X-Admin-Key")) {
		c.JSON(sdk.NewUnauthorized("invalid admin key").AsGinResponse())
		return
	}

	var req RegisterClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(sdk.NewBadRequest("Could not parse request body").WithDetails(err.Error()).AsGinResponse())
		return
	}

	client, err := h.ClientService.CreateClient(req.Name, req.WebhookURL)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, RegisterClientResponse{
		ClientID: int(client.ID),
		APIKey:   client.ApiKey,
	})
}

// Subscribe handles POST /outreach/subscriptions.
func (h *Handler) Subscribe(c *gin.Context) {
	client, ok := h.authorizeClient(c)
	if !ok {
		return
	}

	var req SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(sdk.NewBadRequest("Could not parse request body").WithDetails(err.Error()).AsGinResponse())
		return
	}

	subscription, err := h.SubscriptionService.Subscribe(int(client.ID), req.JobName, req.Priority)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, SubscribeResponse{SubscriptionID: int(subscription.ID)})
}

// Unsubscribe handles DELETE /outreach/subscriptions.
func (h *Handler) Unsubscribe(c *gin.Context) {
	client, ok := h.authorizeClient(c)
	if !ok {
		return
	}

	var req UnsubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(sdk.NewBadRequest("Could not parse request body").WithDetails(err.Error()).AsGinResponse())
		return
	}

	if err := h.SubscriptionService.Unsubscribe(int(client.ID), req.JobName); err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// Helper method to authorize an admin request from a given api key
func (h *Handler) authorizeAdmin(given string) bool {
	if strings.TrimSpace(h.AdminKey) == "" {
		return false
	}
	return strings.TrimSpace(given) == h.AdminKey
}

// Helper method to authorize a client request from a given api key
func (h *Handler) authorizeClient(c *gin.Context) (*domain.Client, bool) {
	apiKey := strings.TrimSpace(c.GetHeader("X-Client-Key"))
	if apiKey == "" {
		c.JSON(sdk.NewUnauthorized("missing client key").AsGinResponse())
		return nil, false
	}

	client, err := h.ClientService.FindByAPIKey(apiKey)
	if err != nil {
		h.handleServiceError(c, err)
		return nil, false
	}

	return client, true
}

// Helper method to handle service errors by attaching them to the current gin context
func (h *Handler) handleServiceError(c *gin.Context, err error) {
	switch {
	case IsErrorCode(err, ErrorValidation):
	case IsErrorCode(err, ErrorConflict):
	case IsErrorCode(err, ErrorNotFound):
		c.JSON(sdk.NewBadRequest(err.Error()).AsGinResponse())
	case IsErrorCode(err, ErrorUnauthorized):
		c.JSON(sdk.NewUnauthorized(err.Error()).AsGinResponse())
	case IsErrorCode(err, ErrorForbidden):
		c.JSON(sdk.NewForbidden().WithDetails(err.Error()).AsGinResponse())
	default:
		c.JSON(sdk.NewInternalServerError("Internal server error").WithDetails(err.Error()).AsGinResponse())
	}
}
