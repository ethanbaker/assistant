package session

import (
	"context"

	"github.com/ethanbaker/assistant/internal/domain"
	"github.com/google/uuid"
	"github.com/nlpodyssey/openai-agents-go/memory"
)

// Repository defines the interface for session storage operations
type Repository interface {
	// CreateSession creates a new session
	CreateSession(ctx context.Context, userID string) (memory.Session, error)

	// GetSession retrieves a session by ID without preloading items
	GetSession(ctx context.Context, sessionID uuid.UUID) (memory.Session, error)

	// GetSessionWithItems retrieves a session by ID with all items preloaded in order
	GetSessionWithItems(ctx context.Context, sessionID uuid.UUID) (memory.Session, error)

	// SaveItem saves an item to the database
	SaveItem(ctx context.Context, item *domain.Item) error

	// GetSessionItems retrieves all items for a session ordered by creation time
	GetSessionItems(ctx context.Context, sessionID uuid.UUID) ([]*domain.Item, error)

	// DeleteSession deletes a session and all its items
	DeleteSession(ctx context.Context, sessionID uuid.UUID) error

	// SearchSessionTranscripts searches session transcripts by query string
	SearchSessionTranscripts(ctx context.Context, query string) ([]*SessionTranscript, error)
}
