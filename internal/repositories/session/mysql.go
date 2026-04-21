package session

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/ethanbaker/assistant/internal/domain"
	"github.com/google/uuid"
	"github.com/nlpodyssey/openai-agents-go/memory"
	"gorm.io/gorm"
)

// MySQLRepository handles session persistence using GORM with MySQL
type MySQLRepository struct {
	db *gorm.DB
}

// NewMySQLRepository creates a new session repository with MySQL connection
func NewMySQLRepository(db *gorm.DB) (*MySQLRepository, error) {
	store := &MySQLRepository{db: db}

	// Auto-migrate tables
	if err := db.AutoMigrate(&SessionEntity{}, &domain.Item{}); err != nil {
		return nil, fmt.Errorf("failed to migrate tables: %w", err)
	}

	return store, nil
}

// CreateSession creates a new session in the database
func (s *MySQLRepository) CreateSession(ctx context.Context, userID string) (memory.Session, error) {
	session := &SessionEntity{
		ID:     uuid.New(),
		UserID: userID,
		mu:     sync.Mutex{},
		Items:  []*domain.Item{},
	}
	session.db = s.db // Set the GORM DB connection

	result := s.db.WithContext(ctx).Create(session)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to create session: %w", result.Error)
	}

	return session, nil
}

// GetSession retrieves a session by ID
func (s *MySQLRepository) GetSession(ctx context.Context, sessionID uuid.UUID) (memory.Session, error) {
	// Get session by ID
	var session SessionEntity
	result := s.db.WithContext(ctx).First(&session, "id = ?", sessionID)

	if result.Error != nil {
		// Handle not found error
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("session not found")
		}
		// Handle generic errors
		return nil, fmt.Errorf("failed to get session: %w", result.Error)
	}

	session.db = s.db // Set the GORM DB connection
	return &session, nil
}

// GetSessionWithItems retrieves a session by ID with all its items preloaded in order
func (s *MySQLRepository) GetSessionWithItems(ctx context.Context, sessionID uuid.UUID) (memory.Session, error) {
	// Get session by ID with items preloaded
	var session SessionEntity
	result := s.db.WithContext(ctx).
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC").Order("id ASC")
		}).
		First(&session, "id = ?", sessionID)

	if result.Error != nil {
		// Handle not found error
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("session not found")
		}
		// Handle generic errors
		return nil, fmt.Errorf("failed to get session with items: %w", result.Error)
	}

	session.db = s.db // Set the GORM DB connection
	return &session, nil
}

// SaveItem saves an item to the database
func (s *MySQLRepository) SaveItem(ctx context.Context, item *domain.Item) error {
	result := s.db.WithContext(ctx).Create(item)
	if result.Error != nil {
		return fmt.Errorf("failed to save item: %w", result.Error)
	}

	return nil
}

// GetSessionItems retrieves all items for a session
func (s *MySQLRepository) GetSessionItems(ctx context.Context, sessionID uuid.UUID) ([]*domain.Item, error) {
	var items []*domain.Item
	result := s.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("created_at ASC").Order("id ASC").Find(&items)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to query items: %w", result.Error)
	}

	return items, nil
}

// DeleteSession deletes a session and its items from the database
func (s *MySQLRepository) DeleteSession(ctx context.Context, sessionID uuid.UUID) error {
	// Start a transaction
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete items associated with the session
		if err := tx.Where("session_id = ?", sessionID).Delete(&domain.Item{}).Error; err != nil {
			return fmt.Errorf("failed to delete session items: %w", err)
		}

		// Delete the session itself
		if err := tx.Where("id = ?", sessionID).Delete(&SessionEntity{}).Error; err != nil {
			return fmt.Errorf("failed to delete session: %w", err)
		}

		return nil
	})
}

// SearchSessionTranscripts performs full-text search across session messages and tool calls
func (s *MySQLRepository) SearchSessionTranscripts(ctx context.Context, query string) ([]*SessionTranscript, error) {
	var transcripts []*SessionTranscript
	searchPattern := "%" + query + "%"

	// Search in items
	var items []domain.Item
	result := s.db.WithContext(ctx).Where("data LIKE ?", searchPattern).
		Order("created_at DESC").Order("id DESC").Limit(50).Find(&items)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to search messages: %w", result.Error)
	}

	// Add items to transcripts
	for _, item := range items {
		// Convert ResponseItem to json
		data, err := json.Marshal(item.ResponseItem)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response item: %w", err)
		}

		transcripts = append(transcripts, &SessionTranscript{
			SessionID: item.SessionID,
			CreatedAt: item.CreatedAt,
			Data:      string(data),
		})
	}

	return transcripts, nil
}

// GetDB returns the underlying GORM database connection
func (s *MySQLRepository) GetDB() *gorm.DB {
	return s.db
}

// Close closes the database connection
func (s *MySQLRepository) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB from gorm.DB: %w", err)
	}
	return sqlDB.Close()
}
