package session

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Session represents a conversation session
type Session struct {
	*gorm.Model
	ID     uuid.UUID `json:"id" gorm:"type:char(36);primaryKey"`
	UserID string    `json:"user_id" gorm:"size:255"`
}

// Message represents a single message in a session
type Message struct {
	*gorm.Model
	ID        uuid.UUID `json:"id" gorm:"type:char(36);primaryKey"`
	SessionID uuid.UUID `json:"session_id" gorm:"type:char(36);not null;index"`
	Role      string    `json:"role" gorm:"size:20;not null"`
	Content   string    `json:"content" gorm:"type:text"`
	Session   Session   `json:"-" gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`
}

// ToolCall represents a tool execution within a session
type ToolCall struct {
	gorm.Model
	ID        uuid.UUID `json:"id" gorm:"type:char(36);primaryKey"`
	SessionID uuid.UUID `json:"session_id" gorm:"type:char(36);not null;index"`
	ToolName  string    `json:"tool_name" gorm:"size:255;not null"`
	Input     string    `json:"input" gorm:"type:text"`
	Output    string    `json:"output" gorm:"type:text"`
	Session   Session   `json:"-" gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`
}

// SessionTranscript represents searchable session content
type SessionTranscript struct {
	SessionID uuid.UUID `json:"session_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// NewSession creates a new session with a generated UUID
func NewSession(userID string) *Session {
	return &Session{
		ID:     uuid.New(),
		UserID: userID,
	}
}

// NewMessage creates a new message with a generated UUID
func NewMessage(sessionID uuid.UUID, role, content string) *Message {
	return &Message{
		ID:        uuid.New(),
		SessionID: sessionID,
		Role:      role,
		Content:   content,
	}
}

// NewToolCall creates a new tool call with a generated UUID
func NewToolCall(sessionID uuid.UUID, toolName, input, output string) *ToolCall {
	return &ToolCall{
		ID:        uuid.New(),
		SessionID: sessionID,
		ToolName:  toolName,
		Input:     input,
		Output:    output,
	}
}
