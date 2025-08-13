package session

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Store handles session persistence using GORM
type Store struct {
	db *gorm.DB
}

// NewStore creates a new session store with GORM connection
func NewStore(databaseURL string) (*Store, error) {
	db, err := gorm.Open(mysql.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &Store{db: db}

	// Auto-migrate tables
	if err := db.AutoMigrate(&Session{}, &Message{}, &ToolCall{}); err != nil {
		return nil, fmt.Errorf("failed to migrate tables: %w", err)
	}

	return store, nil
}

// CreateSession creates a new session in the database
func (s *Store) CreateSession(ctx context.Context, userID string) (*Session, error) {
	session := NewSession(userID)

	result := s.db.WithContext(ctx).Create(session)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to create session: %w", result.Error)
	}

	return session, nil
}

// GetSession retrieves a session by ID
func (s *Store) GetSession(ctx context.Context, sessionID uuid.UUID) (*Session, error) {
	var session Session
	result := s.db.WithContext(ctx).First(&session, "id = ?", sessionID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %w", result.Error)
	}

	return &session, nil
}

// GetSessionWithMessages retrieves a session by ID with all its messages preloaded in order
func (s *Store) GetSessionWithMessages(ctx context.Context, sessionID uuid.UUID) (*Session, error) {
	var session Session
	result := s.db.WithContext(ctx).
		Preload("Messages", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		First(&session, "id = ?", sessionID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session with messages: %w", result.Error)
	}

	return &session, nil
}

// GetSessionWithMessagesAndToolCalls retrieves a session with all messages and their tool calls preloaded
func (s *Store) GetSessionWithMessagesAndToolCalls(ctx context.Context, sessionID uuid.UUID) (*Session, error) {
	var session Session
	result := s.db.WithContext(ctx).
		Preload("Messages", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		Preload("Messages.ToolCalls", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		First(&session, "id = ?", sessionID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session with messages and tool calls: %w", result.Error)
	}

	return &session, nil
}

// SaveMessage saves a message to the database
func (s *Store) SaveMessage(ctx context.Context, message *Message) error {
	result := s.db.WithContext(ctx).Create(message)
	if result.Error != nil {
		return fmt.Errorf("failed to save message: %w", result.Error)
	}

	return nil
}

// SaveToolCall saves a tool call to the database
func (s *Store) SaveToolCall(ctx context.Context, toolCall *ToolCall) error {
	result := s.db.WithContext(ctx).Create(toolCall)
	if result.Error != nil {
		return fmt.Errorf("failed to save tool call: %w", result.Error)
	}

	return nil
}

// GetSessionMessages retrieves all messages for a session
func (s *Store) GetSessionMessages(ctx context.Context, sessionID uuid.UUID) ([]*Message, error) {
	var messages []*Message
	result := s.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("created_at").Find(&messages)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to query messages: %w", result.Error)
	}

	return messages, nil
}

// GetSessionToolCalls retrieves all tool calls for a session by joining with messages
func (s *Store) GetSessionToolCalls(ctx context.Context, sessionID uuid.UUID) ([]*ToolCall, error) {
	var toolCalls []*ToolCall
	result := s.db.WithContext(ctx).
		Joins("JOIN messages ON tool_calls.message_id = messages.id").
		Where("messages.session_id = ?", sessionID).
		Order("tool_calls.created_at").
		Find(&toolCalls)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to query tool calls: %w", result.Error)
	}

	return toolCalls, nil
}

// GetMessageToolCalls retrieves all tool calls for a specific message
func (s *Store) GetMessageToolCalls(ctx context.Context, messageID uint) ([]*ToolCall, error) {
	var toolCalls []*ToolCall
	result := s.db.WithContext(ctx).Where("message_id = ?", messageID).Order("created_at").Find(&toolCalls)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to query message tool calls: %w", result.Error)
	}

	return toolCalls, nil
}

// GetSessionMessagesWithToolCalls retrieves all messages for a session along with their tool calls
func (s *Store) GetSessionMessagesWithToolCalls(ctx context.Context, sessionID uuid.UUID) ([]*Message, error) {
	var messages []*Message
	result := s.db.WithContext(ctx).
		Preload("ToolCalls").
		Where("session_id = ?", sessionID).
		Order("created_at").
		Find(&messages)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to query messages with tool calls: %w", result.Error)
	}

	return messages, nil
}

// SearchSessionTranscripts performs full-text search across session messages and tool calls
func (s *Store) SearchSessionTranscripts(ctx context.Context, query string) ([]*SessionTranscript, error) {
	var transcripts []*SessionTranscript
	searchPattern := "%" + query + "%"

	// Search in messages
	var messages []Message
	result := s.db.WithContext(ctx).Where("content LIKE ?", searchPattern).
		Order("created_at DESC").Limit(50).Find(&messages)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to search messages: %w", result.Error)
	}

	for _, msg := range messages {
		transcripts = append(transcripts, &SessionTranscript{
			SessionID: msg.SessionID,
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt,
		})
	}

	// Search in tool calls (need to join with messages to get session_id)
	var toolCallsWithSession []struct {
		ToolCall
		SessionID uuid.UUID `gorm:"column:session_id"`
	}
	result = s.db.WithContext(ctx).
		Table("tool_calls").
		Select("tool_calls.*, messages.session_id").
		Joins("JOIN messages ON tool_calls.message_id = messages.id").
		Where("tool_calls.input LIKE ? OR tool_calls.output LIKE ?", searchPattern, searchPattern).
		Order("tool_calls.created_at DESC").Limit(50).
		Scan(&toolCallsWithSession)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to search tool calls: %w", result.Error)
	}

	for _, tc := range toolCallsWithSession {
		content := fmt.Sprintf("Tool: %s - Input: %s - Output: %s", tc.ToolCall.ToolName, tc.ToolCall.Input, tc.ToolCall.Output)
		transcripts = append(transcripts, &SessionTranscript{
			SessionID: tc.SessionID,
			Content:   content,
			CreatedAt: tc.ToolCall.CreatedAt,
		})
	}

	return transcripts, nil
}

// GetDB returns the underlying GORM database connection
func (s *Store) GetDB() *gorm.DB {
	return s.db
}

// Close closes the database connection
func (s *Store) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB from gorm.DB: %w", err)
	}
	return sqlDB.Close()
}
