package sdk

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/nlpodyssey/openai-agents-go/memory"
	"gorm.io/gorm"
)

/** Generic Responses */

// Generic success response
type SuccessResponse[T any] struct {
	Code int
	Data T
}

// Return the ErrorResponse in a format to provide to Gin Context
func (r SuccessResponse[T]) AsGinResponse() (int, any) {
	return r.Code, r
}

// Error response
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    int    `json:"code"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Details []any  `json:"details,omitempty"`
}

// Return the ErrorResponse as a marshalable JSON object
func (r ErrorResponse) AsJson() ([]byte, error) {
	return json.Marshal(r)
}

// Return the ErrorResponse in a format to provide to Gin Context
func (r ErrorResponse) AsGinResponse() (int, any) {
	return r.Error.Code, r
}

// WithDetails adds additional details to the error response.
func (e *ErrorResponse) WithDetails(details ...any) *ErrorResponse {
	e.Error.Details = append(e.Error.Details, details...)
	return e
}

// NewSuccess creates a new success response with custom data
func NewSuccess[T any](data T) *SuccessResponse[T] {
	return &SuccessResponse[T]{
		Code: 200,
		Data: data,
	}
}

// NewSuccessMessage creates a new success response with a success message
func NewSuccessMessage(message string) *SuccessResponse[map[string]any] {
	return &SuccessResponse[map[string]any]{
		Code: 200,
		Data: map[string]any{
			"message": message,
		},
	}
}

// NewErrorResponse creates a new custom error response
func NewErrorResponse(code int, status, message string) *ErrorResponse {
	return &ErrorResponse{
		Error: ErrorBody{
			Code:    code,
			Status:  status,
			Message: message,
		},
	}
}

// NewBadRequest creates a new BAD_REQUEST (400) error response with a custom message
func NewBadRequest(message string) *ErrorResponse {
	return &ErrorResponse{
		Error: ErrorBody{
			Code:    400,
			Status:  "BAD_REQUEST",
			Message: message,
		},
	}
}

// NewInternalServerError creates a new INTERNAL_SERVER_ERROR (500) error response with a custom message
func NewInternalServerError(message string) *ErrorResponse {
	return &ErrorResponse{
		Error: ErrorBody{
			Code:    500,
			Status:  "INTERNAL_SERVER_ERROR",
			Message: message,
		},
	}
}

// NewForbidden creates a new FORBIDDEN (403) error response with a custom message
func NewForbidden() *ErrorResponse {
	return &ErrorResponse{
		Error: ErrorBody{
			Code:    403,
			Status:  "FORBIDDEN",
			Message: "Resource is forbidden",
		},
	}
}

// NewUnauthorized creates a new UNAUTHORIZED (401) error response with a custom message
func NewUnauthorized(message string) *ErrorResponse {
	return &ErrorResponse{
		Error: ErrorBody{
			Code:    401,
			Status:  "UNAUTHORIZED",
			Message: "Unauthorized access",
		},
	}
}

/** Requests */

// CreateSessionRequest represents the request body for creating a new session
type CreateSessionRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

// PostMessageRequest represents the request body for adding a message to a session
type PostMessageRequest struct {
	Content string `json:"content" binding:"required"`
	Data    any    `json:"data"`
}

// PostMessageResponse represents the response body after adding a message to a session
type PostMessageResponse struct {
	Items       []Item `json:"items"`
	FinalOutput string `json:"final_output"`
}

// Session represents a user session
type Session struct {
	ID        string         `json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty"`

	UserID string  `json:"user_id"`
	Items  []*Item `json:"items,omitempty"`
}

// ResponseItemData is a wrapper type that implements database serialization
type ResponseItemData struct {
	*memory.TResponseInputItem
}

// MarshalJSON implements custom JSON marshaling for ResponseItemData
func (r ResponseItemData) MarshalJSON() ([]byte, error) {
	if r.TResponseInputItem == nil {
		return json.Marshal(nil)
	}

	// Use a simple map representation to handle the complex union type
	data := map[string]any{
		"content": r.TResponseInputItem,
	}

	return json.Marshal(data)
}

// UnmarshalJSON implements custom JSON unmarshaling for ResponseItemData
func (r *ResponseItemData) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		r.TResponseInputItem = nil
		return nil
	}

	// Try to unmarshal into a generic map first
	var rawData map[string]any
	if err := json.Unmarshal(data, &rawData); err != nil {
		return err
	}

	// Check if it has the expected structure
	if item, exists := rawData["content"]; exists {
		// Re-marshal and unmarshal the item part
		itemBytes, err := json.Marshal(item)
		if err != nil {
			return err
		}

		// Unmarshal into the specific type
		var responseItem memory.TResponseInputItem
		if err := json.Unmarshal(itemBytes, &responseItem); err != nil {
			return err
		}

		r.TResponseInputItem = &responseItem
		return nil
	}

	// If it doesn't have the expected structure, try direct unmarshaling
	var responseItem memory.TResponseInputItem
	if err := json.Unmarshal(data, &responseItem); err != nil {
		// If all else fails, just set to nil to avoid breaking the entire response
		r.TResponseInputItem = nil
		return nil
	}

	r.TResponseInputItem = &responseItem
	return nil
}

// Item represents a message or action within a session
type Item struct {
	ID        uint           `json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at"`

	Data ResponseItemData `json:"data"`

	SessionID uuid.UUID `json:"session_id"`
}

/** Agent Module DTOs */

// AttachJobExecutionContextRequest defines the payload for adding outreach execution context to a session
type AttachJobExecutionContextRequest struct {
	JobExecutionIDs []int `json:"job_execution_ids" binding:"required"`
}

/** Outreach Module DTOs */

// CreateJobRequest is the request payload for creating an outreach job
type CreateJobRequest struct {
	Name       string          `json:"name" binding:"required"`
	Schedule   json.RawMessage `json:"schedule" binding:"required"`
	Handler    string          `json:"handler" binding:"required"`
	Parameters json.RawMessage `json:"parameters"`
	Active     *bool           `json:"active"`
}

// CreateJobResponse is returned after successful job creation
type CreateJobResponse struct {
	JobID int `json:"job_id"`
}

// RegisterClientRequest is the request payload for registering an outreach client
type RegisterClientRequest struct {
	Name       string `json:"name" binding:"required"`
	WebhookURL string `json:"webhook_url" binding:"required"`
}

// RegisterClientResponse is returned after successful client registration
type RegisterClientResponse struct {
	ClientID int    `json:"client_id"`
	APIKey   string `json:"api_key"`
}

// SubscribeRequest is the request payload for subscribing a client to a job
type SubscribeRequest struct {
	JobName  string `json:"job_name" binding:"required"`
	Priority int    `json:"priority" binding:"required"`
}

// SubscribeResponse is returned after successful subscription creation
type SubscribeResponse struct {
	SubscriptionID int `json:"subscription_id"`
}

// UnsubscribeRequest is the request payload for unsubscribing a client from a job
type UnsubscribeRequest struct {
	JobName string `json:"job_name" binding:"required"`
}
