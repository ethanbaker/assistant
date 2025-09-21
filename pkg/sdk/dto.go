package sdk

import (
	"encoding/json"
	"time"

	"github.com/ethanbaker/api/pkg/api_types"
	"github.com/google/uuid"
	"github.com/nlpodyssey/openai-agents-go/memory"
	"gorm.io/gorm"
)

// ApiResponse represents a standard API response structure
type ApiResponse[T any] struct {
	Status  api_types.StatusType `json:"status"`          // Status message
	Code    int                  `json:"code"`            // Status code
	Message string               `json:"message"`         // Human-readable message
	Data    T                    `json:"data,omitempty"`  // Optional data field for successful responses
	Error   any                  `json:"error,omitempty"` // Optional errors field for error responses
}

// AsGinResponse converts the ApiResponse to a format suitable for Gin framework
func (r ApiResponse[T]) AsGinResponse() (int, any) {
	return r.Code, r
}

// AsJSON converts the ApiResponse to a format suitable for JSON responses
func (r ApiResponse[T]) AsJSON() (string, error) {
	b, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func NewSuccess(message string) ApiResponse[any] {
	return ApiResponse[any]{
		Status:  api_types.StatusSuccess,
		Code:    200,
		Message: message,
	}
}

func NewSuccessResponse[T any](message string, data T) ApiResponse[T] {
	return ApiResponse[T]{
		Status:  api_types.StatusSuccess,
		Code:    200,
		Message: message,
		Data:    data,
	}
}

func NewErrorResponse(code int, message string, err any) ApiResponse[any] {
	return ApiResponse[any]{
		Status:  api_types.StatusError,
		Code:    code,
		Message: message,
		Error:   err,
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

// OutreachRegisterRequest represents the request to register an implementation
type OutreachRegisterRequest struct {
	CallbackUrl  string `json:"callback_url" binding:"required"` // HTTP endpoint where outreach requests will be sent
	ClientSecret string `json:"client_secret"`                   // Secret for signing requests (optional)
	ClientId     string `json:"client_id" binding:"required"`    // Unique identifier for the implementation
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
