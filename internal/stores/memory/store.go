package memory

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Store handles memory persistence using GORM
type Store struct {
	db *gorm.DB
}

// NewStore creates a new memory store with GORM connection
func NewStore(databaseURL string) (*Store, error) {
	db, err := gorm.Open(mysql.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &Store{db: db}

	// Auto-migrate tables
	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate tables: %w", err)
	}

	return store, nil
}

// migrate creates or updates the required database tables
func (s *Store) migrate() error {
	return s.db.AutoMigrate(&KeyFact{})
}

// SetFact stores or updates a key fact
func (s *Store) SetFact(ctx context.Context, key, value string) error {
	fact := &KeyFact{
		Key:   key,
		Value: value,
	}

	// GORM's Save will create or update based on primary key
	// For upsert behavior on unique key, we use Create with OnConflict
	result := s.db.WithContext(ctx).Where("fact_key = ?", key).First(&KeyFact{})
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// Create new record
			if err := s.db.WithContext(ctx).Create(fact).Error; err != nil {
				return fmt.Errorf("failed to create fact: %w", err)
			}
		} else {
			// Unexpected error state
			return fmt.Errorf("failed to check existing fact: %w", result.Error)
		}
	} else {
		// Update existing record
		if err := s.db.WithContext(ctx).Model(&KeyFact{}).Where("fact_key = ?", key).Updates(map[string]interface{}{
			"value": value,
		}).Error; err != nil {
			return fmt.Errorf("failed to update fact: %w", err)
		}
	}

	return nil
}

// GetFact retrieves a fact by key
func (s *Store) GetFact(ctx context.Context, key string) (*KeyFact, error) {
	var fact KeyFact
	result := s.db.WithContext(ctx).Where("fact_key = ?", key).First(&fact)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to get fact: %w", result.Error)
	}

	return &fact, nil
}

// SearchFacts searches for facts by key pattern
func (s *Store) SearchFacts(ctx context.Context, pattern string) ([]*KeyFact, error) {
	var facts []*KeyFact
	result := s.db.WithContext(ctx).Where("fact_key LIKE ?", "%"+pattern+"%").Find(&facts)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to search facts: %w", result.Error)
	}

	return facts, nil
}

// ListAllFacts returns all stored facts
func (s *Store) ListAllFacts(ctx context.Context) ([]*KeyFact, error) {
	var facts []*KeyFact
	result := s.db.WithContext(ctx).Order("fact_key").Find(&facts)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list facts: %w", result.Error)
	}

	return facts, nil
}

// DeleteFact removes a fact by key
func (s *Store) DeleteFact(ctx context.Context, key string) error {
	result := s.db.WithContext(ctx).Where("fact_key = ?", key).Delete(&KeyFact{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete fact: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("fact with key '%s' not found", key)
	}

	return nil
}

// Close closes the database connection
func (s *Store) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB from gorm.DB: %w", err)
	}
	return sqlDB.Close()
}

// BuildSearchQuery creates a search query from natural language input
func (s *Store) BuildSearchQuery(input string) string {
	// Simple query building - extract key terms
	words := strings.Fields(strings.ToLower(input))
	var searchTerms []string

	// Filter out common stop words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"what": true, "when": true, "where": true, "why": true, "how": true,
	}

	for _, word := range words {
		if len(word) > 2 && !stopWords[word] {
			searchTerms = append(searchTerms, word)
		}
	}

	if len(searchTerms) == 0 {
		return input // Fallback to original input
	}

	return strings.Join(searchTerms, " ")
}
