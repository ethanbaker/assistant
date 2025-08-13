package session

import (
	"github.com/google/uuid"
	"github.com/nlpodyssey/openai-agents-go/memory"
	"gorm.io/gorm"
)

// Item represents a single item in a session. It can include messages or agent actions
type Item struct {
	*gorm.Model // Base model fields

	ResponseItem *memory.TResponseInputItem `json:"data" gorm:"type:json;not null"` // OpenAI SDK type for response input items

	// Session information
	SessionID uuid.UUID `json:"session_id" gorm:"type:char(36);not null;index"`
	Session   *Session  `json:"-" gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`
}

// NewItem creates a new item
func NewItem(sessionID uuid.UUID, responseItem *memory.TResponseInputItem) *Item {
	return &Item{
		Model:        &gorm.Model{},
		SessionID:    sessionID,
		ResponseItem: responseItem,
	}
}
