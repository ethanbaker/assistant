package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ethanbaker/assistant/internal/domain"
	"github.com/google/uuid"
	"github.com/nlpodyssey/openai-agents-go/memory"
)

// InMemoryRepository provides an in-memory session repository (for testing and one-off operations)
type InMemoryRepository struct {
	sessions map[uuid.UUID]*InMemorySession
	items    map[uuid.UUID][]*domain.Item // sessionID -> items
	mu       sync.RWMutex
}

// NewInMemoryRepository creates a new in-memory session repository
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		sessions: make(map[uuid.UUID]*InMemorySession),
		items:    make(map[uuid.UUID][]*domain.Item),
		mu:       sync.RWMutex{},
	}
}

// CreateSession creates a new session in memory
func (s *InMemoryRepository) CreateSession(ctx context.Context, userID string) (memory.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID := uuid.New()
	session := &InMemorySession{
		ID:        sessionID,
		UserID:    userID,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Items:     []*domain.Item{},
		mu:        sync.RWMutex{},
		store:     s,
	}

	s.sessions[sessionID] = session
	s.items[sessionID] = []*domain.Item{}

	return session, nil
}

// GetSession retrieves a session by ID
func (s *InMemoryRepository) GetSession(ctx context.Context, sessionID uuid.UUID) (memory.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	return session, nil
}

// GetSessionWithItems retrieves a session by ID with all its items preloaded in order
func (s *InMemoryRepository) GetSessionWithItems(ctx context.Context, sessionID uuid.UUID) (memory.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	// Copy items to avoid race conditions
	items := s.items[sessionID]
	session.Items = make([]*domain.Item, len(items))
	copy(session.Items, items)

	return session, nil
}

// SaveItem saves an item to memory
func (s *InMemoryRepository) SaveItem(ctx context.Context, item *domain.Item) error {
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
func (s *InMemoryRepository) GetSessionItems(ctx context.Context, sessionID uuid.UUID) ([]*domain.Item, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items, exists := s.items[sessionID]
	if !exists {
		return []*domain.Item{}, nil
	}

	// Return a copy to avoid race conditions
	result := make([]*domain.Item, len(items))
	copy(result, items)

	return result, nil
}

// DeleteSession deletes a session and its items from memory
func (s *InMemoryRepository) DeleteSession(ctx context.Context, sessionID uuid.UUID) error {
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
func (s *InMemoryRepository) SearchSessionTranscripts(ctx context.Context, query string) ([]*SessionTranscript, error) {
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
