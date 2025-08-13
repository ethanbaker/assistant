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

// SaveMessage saves a message to the database
func (s *Store) SaveMessage(ctx context.Context, message *Message) error {
	if message.ID == uuid.Nil {
		message.ID = uuid.New()
	}

	result := s.db.WithContext(ctx).Create(message)
	if result.Error != nil {
		return fmt.Errorf("failed to save message: %w", result.Error)
	}

	return nil
}

// SaveToolCall saves a tool call to the database
func (s *Store) SaveToolCall(ctx context.Context, toolCall *ToolCall) error {
	if toolCall.ID == uuid.Nil {
		toolCall.ID = uuid.New()
	}

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

// GetSessionToolCalls retrieves all tool calls for a session
func (s *Store) GetSessionToolCalls(ctx context.Context, sessionID uuid.UUID) ([]*ToolCall, error) {
	var toolCalls []*ToolCall
	result := s.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("created_at").Find(&toolCalls)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to query tool calls: %w", result.Error)
	}

	return toolCalls, nil
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

	// Search in tool calls
	var toolCalls []ToolCall
	result = s.db.WithContext(ctx).Where("input LIKE ? OR output LIKE ?", searchPattern, searchPattern).
		Order("created_at DESC").Limit(50).Find(&toolCalls)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to search tool calls: %w", result.Error)
	}

	for _, tc := range toolCalls {
		content := fmt.Sprintf("Tool: %s - Input: %s - Output: %s", tc.ToolName, tc.Input, tc.Output)
		transcripts = append(transcripts, &SessionTranscript{
			SessionID: tc.SessionID,
			Content:   content,
			CreatedAt: tc.CreatedAt,
		})
	}

	return transcripts, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB from gorm.DB: %w", err)
	}
	return sqlDB.Close()
}
