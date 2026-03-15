package session

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/ethanbaker/assistant/internal/domain"
	"github.com/google/uuid"
	"github.com/nlpodyssey/openai-agents-go/memory"
	"github.com/openai/openai-go/v2/shared/constant"
	"gorm.io/gorm"
)

// SessionEntity is the persistence model that implements memory.Session interface
type SessionEntity struct {
	ID        uuid.UUID      `gorm:"type:char(36);primaryKey;unique;not null"`
	CreatedAt time.Time      `gorm:"column:created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
	UserID    string         `gorm:"size:255"`
	Items     []*domain.Item `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`

	db *gorm.DB   `gorm:"-"` // db used in openai-agents-go
	mu sync.Mutex `gorm:"-"` // mutex for thread-safe access
}

// TableName specifies the database table name for GORM
func (*SessionEntity) TableName() string {
	return "sessions"
}

// ToDomain converts entity to domain model
func (e *SessionEntity) ToDomain() *domain.Session {
	return &domain.Session{
		ID:        e.ID,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
		DeletedAt: e.DeletedAt,
		UserID:    e.UserID,
		Items:     e.Items,
	}
}

// FromDomain converts domain model to entity
func FromDomain(s *domain.Session) *SessionEntity {
	return &SessionEntity{
		ID:        s.ID,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
		DeletedAt: s.DeletedAt,
		UserID:    s.UserID,
		Items:     s.Items,
	}
}

// SetDB sets the database connection for the session (for dependency injection)
func (e *SessionEntity) SetDB(db *gorm.DB) {
	e.db = db
}

/** memory.Session interface implementation **/

// SessionID returns the session ID as a string
func (e *SessionEntity) SessionID(ctx context.Context) string {
	return e.ID.String()
}

// GetItems retrieves the conversation history for this session as response input items
// limit is the maximum number of items to retrieve. If <= 0, retrieves all items.
// When specified, returns the latest N items in chronological order.
func (e *SessionEntity) GetItems(ctx context.Context, limit int) ([]memory.TResponseInputItem, error) {
	// Make sure database connection is available
	if e.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	// Query messages associated with this session
	var items []domain.Item
	query := e.db.WithContext(ctx).Where("session_id = ?", e.ID).Order("created_at DESC").Order("id DESC")

	if limit > 0 {
		// Get the latest N messages in descending order first
		query = e.db.WithContext(ctx).Where("session_id = ?", e.ID).Order("created_at DESC").Order("id DESC").Limit(limit)
	}

	// Execute the query
	if err := query.Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve messages: %w", err)
	}

	// Reverse items to chronological order
	for i := 1; i < len(items); i++ {
		key := items[i]
		j := i - 1

		// Items are sorted by CreatedAt ascending, and by ID ascending for tie-breakers
		for j >= 0 && (items[j].CreatedAt.After(key.CreatedAt) || (items[j].CreatedAt.Equal(key.CreatedAt) && items[j].ID > key.ID)) {
			items[j+1] = items[j]
			j--
		}
		items[j+1] = key
	}

	// Convert Item models to TResponseInputItem
	var responseItems []memory.TResponseInputItem
	for _, item := range items {
		if item.ResponseItem.TResponseInputItem != nil {
			responseItems = append(responseItems, *item.ResponseItem.TResponseInputItem)
		}
	}

	// Function calls and call outputs must appear together. So, if the limit ended with a function call output, truncate it
	safeStart := false
	for !safeStart && len(responseItems) > 0 {
		switch *responseItems[0].GetType() {
		case string(constant.ValueOf[constant.FunctionCallOutput]()):
			responseItems = slices.Delete(responseItems, 0, 1)
		case string(constant.ValueOf[constant.ComputerCallOutput]()):
			responseItems = slices.Delete(responseItems, 0, 1)
		case string(constant.ValueOf[constant.LocalShellCallOutput]()):
			responseItems = slices.Delete(responseItems, 0, 1)
		case string(constant.ValueOf[constant.CustomToolCallOutput]()):
			responseItems = slices.Delete(responseItems, 0, 1)
		default:
			safeStart = true
		}
	}

	return responseItems, nil
}

// AddItems adds new items to the conversation history
func (e *SessionEntity) AddItems(ctx context.Context, responseItems []memory.TResponseInputItem) error {
	// Make sure database connection is available
	if e.db == nil {
		return fmt.Errorf("database connection not available")
	}

	// If no response items provided, nothing to add
	if len(responseItems) == 0 {
		return nil
	}

	// Convert TResponseInputItem to Item models
	items := make([]*domain.Item, 0, len(responseItems))
	for _, responseItem := range responseItems {
		items = append(items, &domain.Item{
			SessionID: e.ID,
			CreatedAt: time.Now().UTC(),
			ResponseItem: domain.ResponseItemData{
				TResponseInputItem: &responseItem,
			},
		})
	}

	// Make sure tool calls and their outputs are stored in sequence
	sorted := sortItemsPrePersist(items)

	// Save items to database one-by-one to persist ordering
	tx := e.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	for _, item := range sorted {
		if err := tx.Create(&item).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// PopItem removes and returns the most recent item from the session.
// It returns nil if the session is empty.
func (e *SessionEntity) PopItem(ctx context.Context) (*memory.TResponseInputItem, error) {
	// Make sure database connection is available
	if e.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	var item *domain.Item

	// Find and delete the most recent item in a transaction
	err := e.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Find the most recent item
		if err := tx.Where("session_id = ?", e.ID).Order("created_at DESC").First(&item).Error; err != nil {
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
func (e *SessionEntity) ClearSession(ctx context.Context) error {
	// Make sure database connection is available
	if e.db == nil {
		return fmt.Errorf("database connection not available")
	}

	// Delete all items associated with this session
	if err := e.db.WithContext(ctx).Where("session_id = ?", e.ID).Delete(&domain.Item{}).Error; err != nil {
		return fmt.Errorf("failed to clear session: %w", err)
	}

	return nil
}

// InMemorySession represents a session stored in memory (for testing)
type InMemorySession struct {
	ID        uuid.UUID
	UserID    string
	CreatedAt time.Time
	UpdatedAt time.Time
	Items     []*domain.Item

	mu    sync.RWMutex
	store *InMemoryRepository
}

// ToDomain converts in-memory session to domain model
func (s *InMemorySession) ToDomain() *domain.Session {
	return &domain.Session{
		ID:        s.ID,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
		UserID:    s.UserID,
		Items:     s.Items,
	}
}

/** memory.Session interface implementation for InMemorySession **/

// SessionID returns the session ID as a string
func (s *InMemorySession) SessionID(ctx context.Context) string {
	return s.ID.String()
}

// GetItems retrieves the conversation history for this session as response input items
func (s *InMemorySession) GetItems(ctx context.Context, limit int) ([]memory.TResponseInputItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get items from the store
	items, err := s.store.GetSessionItems(ctx, s.ID)
	if err != nil {
		return nil, err
	}

	// Convert to TResponseInputItem
	var responseItems []memory.TResponseInputItem
	for _, item := range items {
		if item.ResponseItem.TResponseInputItem != nil {
			responseItems = append(responseItems, *item.ResponseItem.TResponseInputItem)
		}
	}

	// Apply limit if specified
	if limit > 0 && len(responseItems) > limit {
		responseItems = responseItems[len(responseItems)-limit:]
	}

	// Function calls and call outputs must appear together
	safeStart := false
	for !safeStart && len(responseItems) > 0 {
		switch *responseItems[0].GetType() {
		case string(constant.ValueOf[constant.FunctionCallOutput]()):
			responseItems = slices.Delete(responseItems, 0, 1)
		case string(constant.ValueOf[constant.ComputerCallOutput]()):
			responseItems = slices.Delete(responseItems, 0, 1)
		case string(constant.ValueOf[constant.LocalShellCallOutput]()):
			responseItems = slices.Delete(responseItems, 0, 1)
		case string(constant.ValueOf[constant.CustomToolCallOutput]()):
			responseItems = slices.Delete(responseItems, 0, 1)
		default:
			safeStart = true
		}
	}

	return responseItems, nil
}

// AddItems adds new items to the conversation history
func (s *InMemorySession) AddItems(ctx context.Context, responseItems []memory.TResponseInputItem) error {
	if len(responseItems) == 0 {
		return nil
	}

	// Convert TResponseInputItem to Item models
	items := make([]*domain.Item, 0, len(responseItems))
	for _, responseItem := range responseItems {
		items = append(items, &domain.Item{
			SessionID: s.ID,
			CreatedAt: time.Now().UTC(),
			ResponseItem: domain.ResponseItemData{
				TResponseInputItem: &responseItem,
			},
		})
	}

	// Make sure tool calls and their outputs are stored in sequence
	for i := 1; i < len(items); i++ {
		prevItemId, prevIsToolCall := getToolCallIdFromInput(items[i-1].ResponseItem)
		currItemId, currIsToolCallOutput := getToolCallIdFromOutput(items[i].ResponseItem)

		if !prevIsToolCall || (currIsToolCallOutput && prevItemId == currItemId) {
			continue
		}

		matchIndex := -1
		for j := i + 1; j < len(items); j++ {
			nextItemId, nextIsToolCallOutput := getToolCallIdFromOutput(items[j].ResponseItem)
			if nextIsToolCallOutput && nextItemId == prevItemId {
				matchIndex = j
				break
			}
		}

		if matchIndex != -1 {
			items[i], items[matchIndex] = items[matchIndex], items[i]
		}
	}

	// Save items to the store
	for _, item := range items {
		if err := s.store.SaveItem(ctx, item); err != nil {
			return err
		}
	}

	return nil
}

// PopItem removes and returns the most recent item from the session
func (s *InMemorySession) PopItem(ctx context.Context) (*memory.TResponseInputItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.Items) == 0 {
		return nil, nil
	}

	// Get the last item
	lastItem := s.Items[len(s.Items)-1]

	// Remove it from the items slice
	s.Items = s.Items[:len(s.Items)-1]

	// Update store
	items, err := s.store.GetSessionItems(ctx, s.ID)
	if err != nil {
		return nil, err
	}

	if len(items) > 0 {
		// Remove the last item from store - this is a simplified implementation
		// In a real scenario, you'd want a DeleteItem method
		s.store.items[s.ID] = items[:len(items)-1]
	}

	return lastItem.ResponseItem.TResponseInputItem, nil
}

// ClearSession clears all items for this session
func (s *InMemorySession) ClearSession(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Items = []*domain.Item{}
	s.store.items[s.ID] = []*domain.Item{}

	return nil
}

// GetItemCount returns the number of items in the session
func (s *InMemorySession) GetItemCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.Items == nil {
		return 0
	}
	return len(s.Items)
}

// GetLastItem returns the last item in the session, or nil if no items exist
func (s *InMemorySession) GetLastItem() *domain.Item {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.Items) == 0 {
		return nil
	}
	return s.Items[len(s.Items)-1]
}

// GetLatestItems returns the latest n items from a session
func (s *InMemorySession) GetLatestItems(ctx context.Context, n int) []domain.Item {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var items []domain.Item

	if n <= 0 || n > len(s.Items) {
		return items
	}

	for i := range n {
		items = append(items, *s.Items[len(s.Items)-1-i])
	}

	return items
}
