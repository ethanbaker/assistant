package keyfact

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ethanbaker/assistant/internal/domain"
)

// InMemoryRepository implements Repository using an in-memory map
type InMemoryRepository struct {
	facts map[string]*domain.KeyFact
	mu    sync.RWMutex
	idSeq uint
}

// NewInMemoryRepository creates a new in-memory repository
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		facts: make(map[string]*domain.KeyFact),
		idSeq: 0,
	}
}

// SetFact stores or updates a key-value fact
func (r *InMemoryRepository) SetFact(ctx context.Context, key, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if fact, exists := r.facts[key]; exists {
		// Update existing fact
		fact.Value = value
		fact.UpdatedAt = time.Now()
	} else {
		// Create new fact
		r.idSeq++
		now := time.Now()
		r.facts[key] = &domain.KeyFact{
			ID:        r.idSeq,
			CreatedAt: now,
			UpdatedAt: now,
			Key:       key,
			Value:     value,
		}
	}

	return nil
}

// GetFact retrieves a fact by key, returns nil if not found
func (r *InMemoryRepository) GetFact(ctx context.Context, key string) (*domain.KeyFact, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	fact, exists := r.facts[key]
	if !exists {
		return nil, nil
	}

	// Return a copy to prevent external modifications
	factCopy := *fact
	return &factCopy, nil
}

// SearchFacts searches for facts by key pattern (LIKE %pattern%)
func (r *InMemoryRepository) SearchFacts(ctx context.Context, pattern string) ([]*domain.KeyFact, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*domain.KeyFact
	lowerPattern := strings.ToLower(pattern)

	for key, fact := range r.facts {
		if fact.DeletedAt.Valid {
			continue // Skip soft-deleted facts
		}
		if strings.Contains(strings.ToLower(key), lowerPattern) {
			factCopy := *fact
			results = append(results, &factCopy)
		}
	}

	return results, nil
}

// ListAllFacts returns all stored facts ordered by key
func (r *InMemoryRepository) ListAllFacts(ctx context.Context) ([]*domain.KeyFact, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*domain.KeyFact

	for _, fact := range r.facts {
		if fact.DeletedAt.Valid {
			continue // Skip soft-deleted facts
		}
		factCopy := *fact
		results = append(results, &factCopy)
	}

	// Sort by key (simple bubble sort for small datasets)
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Key > results[j].Key {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results, nil
}

// DeleteFact removes a fact by key
func (r *InMemoryRepository) DeleteFact(ctx context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	fact, exists := r.facts[key]
	if !exists {
		return fmt.Errorf("fact with key '%s' not found", key)
	}

	// Perform hard delete for in-memory implementation
	delete(r.facts, key)

	// Or use soft delete to match MySQL behavior:
	// fact.DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}

	_ = fact // Keep if using soft delete above

	return nil
}

// Close is a no-op for in-memory implementation
func (r *InMemoryRepository) Close() error {
	// No resources to clean up
	return nil
}
