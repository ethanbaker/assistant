package keyfact

import (
	"context"

	"github.com/ethanbaker/assistant/internal/domain"
)

// Repository defines the interface for memory persistence operations
type Repository interface {
	// SetFact stores or updates a key-value fact
	SetFact(ctx context.Context, key, value string) error

	// GetFact retrieves a fact by key, returns nil if not found
	GetFact(ctx context.Context, key string) (*domain.KeyFact, error)

	// SearchFacts searches for facts by key pattern (LIKE %pattern%)
	SearchFacts(ctx context.Context, pattern string) ([]*domain.KeyFact, error)

	// ListAllFacts returns all stored facts ordered by key
	ListAllFacts(ctx context.Context) ([]*domain.KeyFact, error)

	// DeleteFact removes a fact by key
	DeleteFact(ctx context.Context, key string) error

	// Close closes the underlying database connection
	Close() error
}
