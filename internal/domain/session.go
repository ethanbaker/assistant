package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Session represents the domain model for a conversation session
type Session struct {
	ID        uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
	UserID    string
	Items     []*Item
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

// GetLatestItems returns the latest n items from a session in reverse chronological order
func (s *Session) GetLatestItems(n int) []Item {
	var items []Item

	if n <= 0 || n > len(s.Items) {
		return items
	}

	for i := range n {
		items = append(items, *s.Items[len(s.Items)-1-i])
	}

	return items
}
