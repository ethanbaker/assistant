package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nlpodyssey/openai-agents-go/memory"
	"gorm.io/gorm"
)

// Session represents a conversation session
type Session struct {
	ID        uuid.UUID      `json:"id" gorm:"type:char(36);primaryKey;unique;not null"`
	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"column:deleted_at;index"`

	UserID string  `json:"user_id" gorm:"size:255"`
	Items  []*Item `json:"items,omitempty" gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`

	db *gorm.DB   `json:"-" gorm:"-"` // db used in openai-agents-go
	mu sync.Mutex `json:"-" gorm:"-"` // mutex for thread-safe access
}

/** Message management methods **/

// AddItem adds an item to the session's message list
func (s *Session) AddItem(item *Item) {
	// Set session reference
	item.SessionID = s.ID
	item.Session = s

	s.Items = append(s.Items, item)
}

// GetItemCount returns the number of items in the session
func (s *Session) GetItemCount() int {
	if s.Items == nil {
		return 0
	}
	return len(s.Items)
}

// GetLastItem returns the last item in the session, or nil if no items exist
func (s *Session) GetLastItem() *Item {
	if len(s.Items) == 0 {
		return nil
	}
	return s.Items[len(s.Items)-1]
}

/** memory.Session interface methods **/

// SessionID returns the session ID as a string
func (s *Session) SessionID(ctx context.Context) string {
	return s.ID.String()
}

// GetItems retrieves the conversation history for this session.
// limit is the maximum number of items to retrieve. If <= 0, retrieves all items.
// When specified, returns the latest N items in chronological order.
func (s *Session) GetItems(ctx context.Context, limit int) ([]memory.TResponseInputItem, error) {
	// Make sure database connection is available
	if s.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	// Query messages associated with this session
	var items []Item
	query := s.db.WithContext(ctx).Where("session_id = ?", s.ID).Order("created_at DESC")

	if limit > 0 {
		// Get the latest N messages in descending order first
		query = s.db.WithContext(ctx).Where("session_id = ?", s.ID).Order("created_at DESC").Limit(limit)
	}

	// Execute the query
	if err := query.Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve messages: %w", err)
	}

	// Convert Item models to TResponseInputItem
	var responseItems []memory.TResponseInputItem
	for _, item := range items {
		if item.ResponseItem.TResponseInputItem != nil {
			responseItems = append(responseItems, *item.ResponseItem.TResponseInputItem)
		}
	}

	return responseItems, nil
}

// AddItems adds new items to the conversation history
func (s *Session) AddItems(ctx context.Context, responseItems []memory.TResponseInputItem) error {
	// Make sure database connection is available
	if s.db == nil {
		return fmt.Errorf("database connection not available")
	}

	// If no response items provided, nothing to add
	if len(responseItems) == 0 {
		return nil
	}

	// Convert TResponseInputItem to Item models
	items := make([]*Item, 0, len(responseItems))
	for _, responseItem := range responseItems {
		items = append(items, &Item{
			SessionID: s.ID,
			Session:   s,
			ResponseItem: ResponseItemData{
				TResponseInputItem: &responseItem,
			},
		})
	}

	// Save items to database
	if err := s.db.WithContext(ctx).Create(&items).Error; err != nil {
		return fmt.Errorf("failed to save items: %w", err)
	}

	return nil
}

// PopItem removes and returns the most recent item from the session.
// It returns nil if the session is empty.
func (s *Session) PopItem(ctx context.Context) (*memory.TResponseInputItem, error) {
	// Make sure database connection is available
	if s.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	var item *Item

	// Find and delete the most recent item in a transaction
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Find the most recent item
		if err := tx.Where("session_id = ?", s.ID).Order("created_at DESC").First(&item).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil // No error, just no records
			}
			return err
		}

		// Delete the item
		return tx.Delete(item).Error
	})

	if err != nil {
		return nil, fmt.Errorf("failed to pop item: %w", err)
	}

	// If no item was found, return nil
	if item.ID == 0 {
		return nil, nil
	}

	// Convert to TResponseInputItem
	return item.ResponseItem.TResponseInputItem, nil
}

// ClearSession clears all items for this session.
func (s *Session) ClearSession(ctx context.Context) error {
	// Make sure database connection is available
	if s.db == nil {
		return fmt.Errorf("database connection not available")
	}

	// Delete all items associated with this session
	if err := s.db.WithContext(ctx).Where("session_id = ?", s.ID).Delete(&Item{}).Error; err != nil {
		return fmt.Errorf("failed to clear session: %w", err)
	}

	return nil
}

// SetDB sets the database connection for the session (for dependency injection)
func (s *Session) SetDB(db *gorm.DB) {
	s.db = db
}
