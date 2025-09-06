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
