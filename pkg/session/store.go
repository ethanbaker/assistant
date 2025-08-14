package session

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

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
	if err := db.AutoMigrate(&Session{}, &Item{}); err != nil {
		return nil, fmt.Errorf("failed to migrate tables: %w", err)
	}

	return store, nil
}

// CreateSession creates a new session in the database
func (s *Store) CreateSession(ctx context.Context, userID string) (*Session, error) {
	session := &Session{
		ID:     uuid.New(),
		UserID: userID,
		mu:     sync.Mutex{},
		Items:  []*Item{},
	}
	session.db = s.db // Set the GORM DB connection

	result := s.db.WithContext(ctx).Create(session)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to create session: %w", result.Error)
	}

	return session, nil
}

// GetSession retrieves a session by ID
func (s *Store) GetSession(ctx context.Context, sessionID uuid.UUID) (*Session, error) {
	// Get session by ID
	var session Session
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
func (s *Store) GetSessionWithItems(ctx context.Context, sessionID uuid.UUID) (*Session, error) {
	// Get session by ID with items preloaded
	var session Session
	result := s.db.WithContext(ctx).
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
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

	return &session, nil
}

// SaveItem saves an item to the database
func (s *Store) SaveItem(ctx context.Context, item *Item) error {
	result := s.db.WithContext(ctx).Create(item)
	if result.Error != nil {
		return fmt.Errorf("failed to save item: %w", result.Error)
	}

	return nil
}

// GetSessionItems retrieves all items for a session
func (s *Store) GetSessionItems(ctx context.Context, sessionID uuid.UUID) ([]*Item, error) {
	var items []*Item
	result := s.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("created_at").Find(&items)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to query items: %w", result.Error)
	}

	return items, nil
}

// DeleteSession deletes a session and its items from the database
func (s *Store) DeleteSession(ctx context.Context, sessionID uuid.UUID) error {
	// Start a transaction
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete items associated with the session
		if err := tx.Where("session_id = ?", sessionID).Delete(&Item{}).Error; err != nil {
			return fmt.Errorf("failed to delete session items: %w", err)
		}

		// Delete the session itself
		if err := tx.Where("id = ?", sessionID).Delete(&Session{}).Error; err != nil {
			return fmt.Errorf("failed to delete session: %w", err)
		}

		return nil
	})
}

// SearchSessionTranscripts performs full-text search across session messages and tool calls
func (s *Store) SearchSessionTranscripts(ctx context.Context, query string) ([]*SessionTranscript, error) {
	var transcripts []*SessionTranscript
	searchPattern := "%" + query + "%"

	// Search in items
	var items []Item
	result := s.db.WithContext(ctx).Where("data LIKE ?", searchPattern).
		Order("created_at DESC").Limit(50).Find(&items)
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
