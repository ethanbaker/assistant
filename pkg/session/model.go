package session

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/nlpodyssey/openai-agents-go/memory"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/responses"
	"gorm.io/gorm"
)

// Session represents a conversation session
type Session struct {
	*gorm.Model

	ID       uuid.UUID `json:"id" gorm:"type:char(36);primaryKey"`
	UserID   string    `json:"user_id" gorm:"size:255"`
	Messages []Message `json:"messages,omitempty" gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`

	// DB access for interface methods (optional, can be injected)
	db *gorm.DB `json:"-" gorm:"-"`
}

// Message represents a single message in a session
type Message struct {
	*gorm.Model

	Role      string     `json:"role" gorm:"size:20;not null"`
	Content   string     `json:"content" gorm:"type:text"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty" gorm:"foreignKey:MessageID;constraint:OnDelete:CASCADE"`

	SessionID uuid.UUID `json:"session_id" gorm:"type:char(36);not null;index"`
	Session   Session   `json:"-" gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`
}

// ToolCall represents a tool execution within a message
type ToolCall struct {
	*gorm.Model

	ToolName string `json:"tool_name" gorm:"size:255;not null"`
	Input    string `json:"input" gorm:"type:text"`
	Output   string `json:"output" gorm:"type:text"`

	MessageID uint    `json:"message_id" gorm:"not null;index"`
	Message   Message `json:"-" gorm:"foreignKey:MessageID;constraint:OnDelete:CASCADE"`
}

// SessionTranscript represents searchable session content
type SessionTranscript struct {
	SessionID uuid.UUID `json:"session_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// AddMessage adds a message to the session's message list
func (s *Session) AddMessage(message *Message) {
	if s.Messages == nil {
		s.Messages = make([]Message, 0)
	}
	message.SessionID = s.ID
	s.Messages = append(s.Messages, *message)
}

// GetMessageCount returns the number of messages in the session
func (s *Session) GetMessageCount() int {
	return len(s.Messages)
}

// GetLastMessage returns the last message in the session, or nil if no messages exist
func (s *Session) GetLastMessage() *Message {
	if len(s.Messages) == 0 {
		return nil
	}
	return &s.Messages[len(s.Messages)-1]
}

// NewSession creates a new session with a generated UUID
func NewSession(userID string) *Session {
	return &Session{
		Model:  &gorm.Model{},
		ID:     uuid.New(),
		UserID: userID,
	}
}

// NewMessage creates a new message
func NewMessage(sessionID uuid.UUID, role, content string) *Message {
	return &Message{
		Model:     &gorm.Model{},
		SessionID: sessionID,
		Role:      role,
		Content:   content,
	}
}

// NewToolCall creates a new tool call
func NewToolCall(messageID uint, toolName, input, output string) *ToolCall {
	return &ToolCall{
		Model:     &gorm.Model{},
		MessageID: messageID,
		ToolName:  toolName,
		Input:     input,
		Output:    output,
	}
}

// memory.Session interface implementation

// SessionID returns the session ID as a string
func (s *Session) SessionID(ctx context.Context) string {
	return s.ID.String()
}

// GetItems retrieves the conversation history for this session.
// limit is the maximum number of items to retrieve. If <= 0, retrieves all items.
// When specified, returns the latest N items in chronological order.
func (s *Session) GetItems(ctx context.Context, limit int) ([]memory.TResponseInputItem, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	var messages []Message
	query := s.db.WithContext(ctx).Where("session_id = ?", s.ID).Order("created_at ASC")

	if limit > 0 {
		// Get the latest N messages in descending order first
		query = s.db.WithContext(ctx).Where("session_id = ?", s.ID).Order("created_at DESC").Limit(limit)
	}

	if err := query.Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve messages: %w", err)
	}

	// If we used limit, reverse to get chronological order
	if limit > 0 {
		slices.Reverse(messages)
	}

	// Convert messages to TResponseInputItem
	items := make([]memory.TResponseInputItem, 0, len(messages))
	for _, msg := range messages {
		item, err := s.messageToInputItem(msg)
		if err != nil {
			// Skip invalid messages rather than failing the whole request
			continue
		}
		items = append(items, item)
	}

	return items, nil
}

// AddItems adds new items to the conversation history.
func (s *Session) AddItems(ctx context.Context, items []memory.TResponseInputItem) error {
	if s.db == nil {
		return fmt.Errorf("database connection not available")
	}

	if len(items) == 0 {
		return nil
	}

	// Convert TResponseInputItem to Message models
	messages := make([]Message, 0, len(items))
	for _, item := range items {
		msg, err := s.inputItemToMessage(item)
		if err != nil {
			return fmt.Errorf("failed to convert input item to message: %w", err)
		}
		msg.SessionID = s.ID
		messages = append(messages, msg)
	}

	// Save messages to database
	if err := s.db.WithContext(ctx).Create(&messages).Error; err != nil {
		return fmt.Errorf("failed to save messages: %w", err)
	}

	return nil
}

// PopItem removes and returns the most recent item from the session.
// It returns nil if the session is empty.
func (s *Session) PopItem(ctx context.Context) (*memory.TResponseInputItem, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	var message Message

	// Find and delete the most recent message in a transaction
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Find the most recent message
		if err := tx.Where("session_id = ?", s.ID).Order("created_at DESC").First(&message).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil // No error, just no records
			}
			return err
		}

		// Delete the message
		return tx.Delete(&message).Error
	})

	if err != nil {
		return nil, fmt.Errorf("failed to pop message: %w", err)
	}

	// If no message was found, return nil
	if message.ID == 0 {
		return nil, nil
	}

	// Convert to TResponseInputItem
	item, err := s.messageToInputItem(message)
	if err != nil {
		return nil, fmt.Errorf("failed to convert message to input item: %w", err)
	}

	return &item, nil
}

// ClearSession clears all items for this session.
func (s *Session) ClearSession(ctx context.Context) error {
	if s.db == nil {
		return fmt.Errorf("database connection not available")
	}

	if err := s.db.WithContext(ctx).Where("session_id = ?", s.ID).Delete(&Message{}).Error; err != nil {
		return fmt.Errorf("failed to clear session: %w", err)
	}

	return nil
}

// SetDB sets the database connection for the session (for dependency injection)
func (s *Session) SetDB(db *gorm.DB) {
	s.db = db
}

// Helper methods for conversion between Message and TResponseInputItem

// messageToInputItem converts a Message to TResponseInputItem
func (s *Session) messageToInputItem(msg Message) (memory.TResponseInputItem, error) {
	// Try to unmarshal as stored JSON first (for messages that were originally TResponseInputItem)
	if msg.Content != "" && (msg.Content[0] == '{' || msg.Content[0] == '[') {
		var item memory.TResponseInputItem
		if err := json.Unmarshal([]byte(msg.Content), &item); err == nil {
			// Successfully unmarshaled as TResponseInputItem
			return item, nil
		}
	}

	// Convert simple message to TResponseInputItem
	role := s.roleStringToInputRole(msg.Role)

	return memory.TResponseInputItem{
		OfMessage: &responses.EasyInputMessageParam{
			Content: responses.EasyInputMessageContentUnionParam{
				OfString: param.NewOpt(msg.Content),
			},
			Role: role,
			Type: responses.EasyInputMessageTypeMessage,
		},
	}, nil
}

// inputItemToMessage converts TResponseInputItem to Message
func (s *Session) inputItemToMessage(item memory.TResponseInputItem) (Message, error) {
	// Handle different types of input items
	if easyMsg := item.OfMessage; !param.IsOmitted(easyMsg) {
		// Simple message
		content := ""
		if !param.IsOmitted(easyMsg.Content.OfString) {
			content = easyMsg.Content.OfString.Value
		} else {
			// For more complex content, store as JSON
			jsonBytes, err := json.Marshal(item)
			if err != nil {
				return Message{}, fmt.Errorf("failed to marshal complex message: %w", err)
			}
			content = string(jsonBytes)
		}

		return Message{
			Role:    s.inputRoleToRoleString(easyMsg.Role),
			Content: content,
		}, nil
	} else {
		// For non-message items (function calls, etc.), store as JSON
		jsonBytes, err := json.Marshal(item)
		if err != nil {
			return Message{}, fmt.Errorf("failed to marshal input item: %w", err)
		}

		return Message{
			Role:    "system", // Use system role for non-message items
			Content: string(jsonBytes),
		}, nil
	}
}

// roleStringToInputRole converts string role to responses.EasyInputMessageRole
func (s *Session) roleStringToInputRole(role string) responses.EasyInputMessageRole {
	switch role {
	case "user":
		return responses.EasyInputMessageRoleUser
	case "assistant":
		return responses.EasyInputMessageRoleAssistant
	case "system":
		return responses.EasyInputMessageRoleSystem
	case "developer":
		return responses.EasyInputMessageRoleDeveloper
	default:
		return responses.EasyInputMessageRoleUser // Default to user
	}
}

// inputRoleToRoleString converts responses.EasyInputMessageRole to string
func (s *Session) inputRoleToRoleString(role responses.EasyInputMessageRole) string {
	switch role {
	case responses.EasyInputMessageRoleUser:
		return "user"
	case responses.EasyInputMessageRoleAssistant:
		return "assistant"
	case responses.EasyInputMessageRoleSystem:
		return "system"
	case responses.EasyInputMessageRoleDeveloper:
		return "developer"
	default:
		return "user" // Default to user
	}
}
