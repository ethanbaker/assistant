package keyfact

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethanbaker/assistant/internal/domain"
	"gorm.io/gorm"
)

// MySQLRepository handles memory persistence using GORM with MySQL
type MySQLRepository struct {
	db *gorm.DB
}

// NewMySQLRepository creates a new memory repository with MySQL connection
func NewMySQLRepository(db *gorm.DB) (*MySQLRepository, error) {
	store := &MySQLRepository{db: db}

	// Auto-migrate tables
	if err := db.AutoMigrate(&domain.KeyFact{}); err != nil {
		return nil, fmt.Errorf("failed to migrate tables: %w", err)
	}

	return store, nil
}

// SetFact stores or updates a key fact
func (r *MySQLRepository) SetFact(ctx context.Context, key, value string) error {
	fact := &domain.KeyFact{
		Key:   key,
		Value: value,
	}

	// GORM's Save will create or update based on primary key
	// For upsert behavior on unique key, we use Create with OnConflict
	result := r.db.WithContext(ctx).Where("fact_key = ?", key).First(&domain.KeyFact{})
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// Create new record
			if err := r.db.WithContext(ctx).Create(fact).Error; err != nil {
				return fmt.Errorf("failed to create fact: %w", err)
			}
		} else {
			// Unexpected error state
			return fmt.Errorf("failed to check existing fact: %w", result.Error)
		}
	} else {
		// Update existing record
		if err := r.db.WithContext(ctx).Model(&domain.KeyFact{}).Where("fact_key = ?", key).Updates(map[string]interface{}{
			"value": value,
		}).Error; err != nil {
			return fmt.Errorf("failed to update fact: %w", err)
		}
	}

	return nil
}

// GetFact retrieves a fact by key
func (r *MySQLRepository) GetFact(ctx context.Context, key string) (*domain.KeyFact, error) {
	var fact domain.KeyFact
	result := r.db.WithContext(ctx).Where("fact_key = ?", key).First(&fact)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to get fact: %w", result.Error)
	}

	return &fact, nil
}

// SearchFacts searches for facts by key pattern
func (r *MySQLRepository) SearchFacts(ctx context.Context, pattern string) ([]*domain.KeyFact, error) {
	var facts []*domain.KeyFact
	result := r.db.WithContext(ctx).Where("fact_key LIKE ?", "%"+pattern+"%").Find(&facts)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to search facts: %w", result.Error)
	}

	return facts, nil
}

// ListAllFacts returns all stored facts
func (r *MySQLRepository) ListAllFacts(ctx context.Context) ([]*domain.KeyFact, error) {
	var facts []*domain.KeyFact
	result := r.db.WithContext(ctx).Order("fact_key").Find(&facts)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list facts: %w", result.Error)
	}

	return facts, nil
}

// DeleteFact removes a fact by key
func (r *MySQLRepository) DeleteFact(ctx context.Context, key string) error {
	result := r.db.WithContext(ctx).Where("fact_key = ?", key).Delete(&domain.KeyFact{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete fact: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("fact with key '%s' not found", key)
	}

	return nil
}

// Close closes the database connection
func (r *MySQLRepository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB from gorm.DB: %w", err)
	}
	return sqlDB.Close()
}

// buildSearchQuery creates a search query from natural language input
func buildSearchQuery(input string) string {
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
