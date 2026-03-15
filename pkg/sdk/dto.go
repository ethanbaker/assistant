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
	return r.Code, r.Data
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

/** Outreach Module DTOs */

// OutreachCredentials represents credentials for outreach implementations
type OutreachCredentials struct {
	ClientId     string `json:"client_id" binding:"required"`     // Unique identifier for the implementation
	ClientSecret string `json:"client_secret" binding:"required"` // Secret for signing requests
}

// OutreachRegisterRequest represents the request to register an implementation
type OutreachRegisterRequest struct {
	CallbackUrl  string `json:"callback_url" binding:"required"`  // HTTP endpoint where outreach requests will be sent
	ClientId     string `json:"client_id" binding:"required"`     // Unique identifier for the implementation
	ClientSecret string `json:"client_secret" binding:"required"` // Secret for signing requests
}

// OutreachRegisterResponse represents the successful registration response
type OutreachRegisterResponse struct {
	ClientId string `json:"client_id"` // The registered client ID
}

// OutreachUnregisterRequest represents the request to unregister an implementation
type OutreachUnregisterRequest struct {
	ClientId string `json:"client_id" binding:"required"` // Client ID to unregister
}

// OutreachImplementation represents an implementation in API responses
type OutreachImplementation struct {
	ClientId    string `json:"client_id"`    // Unique identifier for the implementation
	CallbackUrl string `json:"callback_url"` // HTTP endpoint where outreach requests will be sent
}

// OutreachListImplementationsResponse represents the response for listing implementations
type OutreachListImplementationsResponse struct {
	Implementations []OutreachImplementation `json:"implementations"`
	Count           int                      `json:"count"`
}

// OutreachTaskStatus represents the status of task operations
type OutreachTaskStatus struct {
	Loaded int `json:"loaded"` // Number of tasks loaded
}

// OutreachStatusResponse represents the overall status of the outreach service
type OutreachStatusResponse struct {
	Status               string             `json:"status"`                // Overall service status
	TasksStatus          OutreachTaskStatus `json:"tasks_status"`          // Task statistics
	ImplementationsCount int                `json:"implementations_count"` // Number of registered implementations
	ManagerRunning       bool               `json:"manager_running"`       // Whether the manager is running
}

// OutreachResponseRequest sent by the outreach service to an implementation
// This represents the payload that will be sent to registered implementations
type OutreachRequest struct {
	Id     string         `json:"id"`               // Idempotency ID for the request
	Author string         `json:"author,omitempty"` // Implementation author of the request
	Key    string         `json:"key"`              // Name of the outreach task being performed
	Params map[string]any `json:"params"`           // Task parameters from the original task

	Content string `json:"content"`        // Content to be sent out (generated by the task)
	Data    any    `json:"data,omitempty"` // Extra data for the request
}
