package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Store interface defines methods for session storage
type Store interface {
	CreateSession(ctx context.Context, userID string) (Session, error)
	GetSession(ctx context.Context, sessionID uuid.UUID) (Session, error)
	GetSessionWithItems(ctx context.Context, sessionID uuid.UUID) (Session, error)
	SaveItem(ctx context.Context, item *Item) error
	GetSessionItems(ctx context.Context, sessionID uuid.UUID) ([]*Item, error)
	DeleteSession(ctx context.Context, sessionID uuid.UUID) error
	SearchSessionTranscripts(ctx context.Context, query string) ([]*SessionTranscript, error)
}

// MySqlStore handles session persistence using GORM
type MySqlStore struct {
	db *gorm.DB
}

// NewMySqlStore creates a new session store with GORM connection
func NewMySqlStore(databaseURL string) (*MySqlStore, error) {
	db, err := gorm.Open(mysql.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &MySqlStore{db: db}

	// Auto-migrate tables
	if err := db.AutoMigrate(&MySqlSession{}, &Item{}); err != nil {
		return nil, fmt.Errorf("failed to migrate tables: %w", err)
	}

	return store, nil
}

// CreateSession creates a new session in the database
func (s *MySqlStore) CreateSession(ctx context.Context, userID string) (Session, error) {
	session := &MySqlSession{
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
func (s *MySqlStore) GetSession(ctx context.Context, sessionID uuid.UUID) (Session, error) {
	// Get session by ID
	var session MySqlSession
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
func (s *MySqlStore) GetSessionWithItems(ctx context.Context, sessionID uuid.UUID) (Session, error) {
	// Get session by ID with items preloaded
	var session MySqlSession
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

	return &session, nil
}

// SaveItem saves an item to the database
func (s *MySqlStore) SaveItem(ctx context.Context, item *Item) error {
	result := s.db.WithContext(ctx).Create(item)
	if result.Error != nil {
		return fmt.Errorf("failed to save item: %w", result.Error)
	}

	return nil
}

// GetSessionItems retrieves all items for a session
func (s *MySqlStore) GetSessionItems(ctx context.Context, sessionID uuid.UUID) ([]*Item, error) {
	var items []*Item
	result := s.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("created_at ASC").Order("id ASC").Find(&items)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to query items: %w", result.Error)
	}

	return items, nil
}

// DeleteSession deletes a session and its items from the database
func (s *MySqlStore) DeleteSession(ctx context.Context, sessionID uuid.UUID) error {
	// Start a transaction
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete items associated with the session
		if err := tx.Where("session_id = ?", sessionID).Delete(&Item{}).Error; err != nil {
			return fmt.Errorf("failed to delete session items: %w", err)
		}

		// Delete the session itself
		if err := tx.Where("id = ?", sessionID).Delete(&MySqlSession{}).Error; err != nil {
			return fmt.Errorf("failed to delete session: %w", err)
		}

		return nil
	})
}

// SearchSessionTranscripts performs full-text search across session messages and tool calls
func (s *MySqlStore) SearchSessionTranscripts(ctx context.Context, query string) ([]*SessionTranscript, error) {
	var transcripts []*SessionTranscript
	searchPattern := "%" + query + "%"

	// Search in items
	var items []Item
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
func (s *MySqlStore) GetDB() *gorm.DB {
	return s.db
}

// Close closes the database connection
func (s *MySqlStore) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB from gorm.DB: %w", err)
	}
	return sqlDB.Close()
}

// InMemoryStore creates a new in-memory session store (for one-off operations)
type InMemoryStore struct {
	sessions map[uuid.UUID]*InMemorySession
	items    map[uuid.UUID][]*Item // sessionID -> items
	mu       sync.RWMutex
}

// NewInMemoryStore creates a new in-memory session store
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		sessions: make(map[uuid.UUID]*InMemorySession),
		items:    make(map[uuid.UUID][]*Item),
		mu:       sync.RWMutex{},
	}
}

// CreateSession creates a new session in memory
func (s *InMemoryStore) CreateSession(ctx context.Context, userID string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID := uuid.New()
	session := &InMemorySession{
		ID:        sessionID,
		UserID:    userID,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Items:     []*Item{},
		mu:        sync.RWMutex{},
		store:     s,
	}

	s.sessions[sessionID] = session
	s.items[sessionID] = []*Item{}

	return session, nil
}

// GetSession retrieves a session by ID
func (s *InMemoryStore) GetSession(ctx context.Context, sessionID uuid.UUID) (Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	return session, nil
}

// GetSessionWithItems retrieves a session by ID with all its items preloaded in order
func (s *InMemoryStore) GetSessionWithItems(ctx context.Context, sessionID uuid.UUID) (Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	// Copy items to avoid race conditions
	items := s.items[sessionID]
	session.Items = make([]*Item, len(items))
	copy(session.Items, items)

	return session, nil
}

// SaveItem saves an item to memory
func (s *InMemoryStore) SaveItem(ctx context.Context, item *Item) error {
	if item == nil {
		return fmt.Errorf("item cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if session exists
	if _, exists := s.sessions[item.SessionID]; !exists {
		return fmt.Errorf("session not found")
	}

	// Set timestamps
	now := time.Now().UTC()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now

	// Generate ID if not set
	if item.ID == 0 {
		// Simple ID generation - use the length as ID
		item.ID = uint(len(s.items[item.SessionID]) + 1)
	}

	// Add item to session
	s.items[item.SessionID] = append(s.items[item.SessionID], item)

	// Update session items reference
	if session, exists := s.sessions[item.SessionID]; exists {
		session.Items = append(session.Items, item)
	}

	return nil
}

// GetSessionItems retrieves all items for a session
func (s *InMemoryStore) GetSessionItems(ctx context.Context, sessionID uuid.UUID) ([]*Item, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items, exists := s.items[sessionID]
	if !exists {
		return []*Item{}, nil
	}

	// Return a copy to avoid race conditions
	result := make([]*Item, len(items))
	copy(result, items)

	return result, nil
}

// DeleteSession deletes a session and its items from memory
func (s *InMemoryStore) DeleteSession(ctx context.Context, sessionID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[sessionID]; !exists {
		return fmt.Errorf("session not found")
	}

	// Delete session and its items
	delete(s.sessions, sessionID)
	delete(s.items, sessionID)

	return nil
}

// SearchSessionTranscripts performs full-text search across session messages and tool calls
func (s *InMemoryStore) SearchSessionTranscripts(ctx context.Context, query string) ([]*SessionTranscript, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var transcripts []*SessionTranscript

	// Search through all items in all sessions
	for sessionID, items := range s.items {
		for _, item := range items {
			// Convert ResponseItem to json for searching
			data, err := json.Marshal(item.ResponseItem)
			if err != nil {
				continue // Skip items that can't be marshaled
			}

			// Check if query matches the data
			dataStr := string(data)
			if contains(dataStr, query) {
				transcripts = append(transcripts, &SessionTranscript{
					SessionID: sessionID,
					CreatedAt: item.CreatedAt,
					Data:      dataStr,
				})
			}
		}
	}

	// Sort transcripts by created time (descending)
	for i := 1; i < len(transcripts); i++ {
		key := transcripts[i]
		j := i - 1

		for j >= 0 && transcripts[j].CreatedAt.Before(key.CreatedAt) {
			transcripts[j+1] = transcripts[j]
			j--
		}
		transcripts[j+1] = key
	}

	// Limit to 50 results like MySQL version
	if len(transcripts) > 50 {
		transcripts = transcripts[:50]
	}

	return transcripts, nil
}

// contains performs case-insensitive substring search
func contains(s, substr string) bool {
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	return strings.Contains(s, substr)
}
