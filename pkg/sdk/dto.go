package sdk

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
	Output any `json:"output"`
}
