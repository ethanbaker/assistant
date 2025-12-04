package session

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nlpodyssey/openai-agents-go/memory"
	"github.com/openai/openai-go/v2/shared/constant"
	"gorm.io/gorm"
)

// Sessions implement the memory.Session interface and other helpful methods
type Session interface {
	memory.Session

	GetItemCount() int
	GetLastItem() *Item
	GetLatestItems(ctx context.Context, n int) []Item
}

// MySqlSession represents a conversation session
type MySqlSession struct {
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

// GetItemCount returns the number of items in the session
func (s *MySqlSession) GetItemCount() int {
	if s.Items == nil {
		return 0
	}
	return len(s.Items)
}

// GetLastItem returns the last item in the session, or nil if no items exist
func (s *MySqlSession) GetLastItem() *Item {
	if len(s.Items) == 0 {
		return nil
	}
	return s.Items[len(s.Items)-1]
}

// Get the latest n items from a session
func (s *MySqlSession) GetLatestItems(ctx context.Context, n int) []Item {
	var items []Item

	if n <= 0 || n > len(s.Items) {
		return items
	}

	for i := range n {
		items = append(items, *s.Items[len(s.Items)-1-i])
	}

	return items
}

/** External memory.Session interface methods **/

// SessionID returns the session ID as a string
func (s *MySqlSession) SessionID(ctx context.Context) string {
	return s.ID.String()
}

// GetItems retrieves the conversation history for this session as response input items
// limit is the maximum number of items to retrieve. If <= 0, retrieves all items.
// When specified, returns the latest N items in chronological order.
func (s *MySqlSession) GetItems(ctx context.Context, limit int) ([]memory.TResponseInputItem, error) {
	// Make sure database connection is available
	if s.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	// Query messages associated with this session
	var items []Item
	query := s.db.WithContext(ctx).Where("session_id = ?", s.ID).Order("created_at DESC").Order("id DESC")

	if limit > 0 {
		// Get the latest N messages in descending order first
		query = s.db.WithContext(ctx).Where("session_id = ?", s.ID).Order("created_at DESC").Order("id DESC").Limit(limit)
	}

	// Execute the query
	if err := query.Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve messages: %w", err)
	}

	// Reverse items to chronological order
	// We perform a longer insertion sort just to be safe as
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
func (s *MySqlSession) AddItems(ctx context.Context, responseItems []memory.TResponseInputItem) error {
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
			CreatedAt: time.Now().UTC(),
			ResponseItem: ResponseItemData{
				TResponseInputItem: &responseItem,
			},
		})
	}

	// Make sure tool calls and their outputs are stored in sequence
	for i := 1; i < len(items); i++ {
		prevItemId, prevIsToolCall := getToolCallIdFromInput(items[i-1].ResponseItem)
		currItemId, currIsToolCallOutput := getToolCallIdFromOutput(items[i].ResponseItem)

		// We don't care about current comparison if previous item is not a tool call or if it is a tool call in valid sequence
		if !prevIsToolCall || (currIsToolCallOutput && prevItemId == currItemId) {
			continue
		}

		// Make sure the corresponding tool call output is next
		matchIndex := -1
		for j := i + 1; j < len(items); j++ {
			nextItemId, nextIsToolCallOutput := getToolCallIdFromOutput(items[j].ResponseItem)
			if nextIsToolCallOutput && nextItemId == prevItemId {
				matchIndex = j
				break
			}
		}

		if matchIndex != -1 {
			// We found a matching tool call output, reorder the items
			items[i], items[matchIndex] = items[matchIndex], items[i]
		}

	}

	// Save items to database one-by-one to persist ordering
	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	for _, item := range items {
		if err := tx.Create(&item).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// PopItem removes and returns the most recent item from the session.
// It returns nil if the session is empty.
func (s *MySqlSession) PopItem(ctx context.Context) (*memory.TResponseInputItem, error) {
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
func (s *MySqlSession) ClearSession(ctx context.Context) error {
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

// TableName specifies the database table name for GORM
func (*MySqlSession) TableName() string {
	return "sessions"
}

// SetDB sets the database connection for the session (for dependency injection)
func (s *MySqlSession) SetDB(db *gorm.DB) {
	s.db = db
}

// InMemorySession represents a conversation session stored in memory
type InMemorySession struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	UserID string  `json:"user_id"`
	Items  []*Item `json:"items,omitempty"`

	store *InMemoryStore `json:"-"` // reference to the store for database operations
	mu    sync.RWMutex   `json:"-"` // mutex for thread-safe access
}

/** Message management methods **/

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
func (s *InMemorySession) GetLastItem() *Item {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.Items) == 0 {
		return nil
	}
	return s.Items[len(s.Items)-1]
}

// GetLatestItems gets the latest n items from a session
func (s *InMemorySession) GetLatestItems(ctx context.Context, n int) []Item {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var items []Item

	if n <= 0 || n > len(s.Items) {
		return items
	}

	for i := range n {
		items = append(items, *s.Items[len(s.Items)-1-i])
	}

	return items
}

/** External memory.Session interface methods **/

// SessionID returns the session ID as a string
func (s *InMemorySession) SessionID(ctx context.Context) string {
	return s.ID.String()
}

// GetItems retrieves the conversation history for this session as response input items
// limit is the maximum number of items to retrieve. If <= 0, retrieves all items.
// When specified, returns the latest N items in chronological order.
func (s *InMemorySession) GetItems(ctx context.Context, limit int) ([]memory.TResponseInputItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get items from the store
	items, err := s.store.GetSessionItems(ctx, s.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve items: %w", err)
	}

	// Sort items in descending order first if we need to limit
	if limit > 0 && len(items) > limit {
		// Get the latest N items
		items = items[len(items)-limit:]
	}

	// Sort items chronologically (ascending by CreatedAt, then by ID)
	for i := 1; i < len(items); i++ {
		key := items[i]
		j := i - 1

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

	return responseItems, nil
}

// AddItems adds new items to the conversation history
func (s *InMemorySession) AddItems(ctx context.Context, responseItems []memory.TResponseInputItem) error {
	if s.store == nil {
		return fmt.Errorf("store connection not available")
	}

	// If no response items provided, nothing to add
	if len(responseItems) == 0 {
		return nil
	}

	// Convert TResponseInputItem to Item models
	for _, responseItem := range responseItems {
		item := &Item{
			SessionID: s.ID,
			CreatedAt: time.Now().UTC(),
			ResponseItem: ResponseItemData{
				TResponseInputItem: &responseItem,
			},
		}

		// Save item to store
		if err := s.store.SaveItem(ctx, item); err != nil {
			return fmt.Errorf("failed to save item: %w", err)
		}
	}

	return nil
}

// PopItem removes and returns the most recent item from the session.
// It returns nil if the session is empty.
func (s *InMemorySession) PopItem(ctx context.Context) (*memory.TResponseInputItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.store == nil {
		return nil, fmt.Errorf("store connection not available")
	}

	// Get items from store
	items, err := s.store.GetSessionItems(ctx, s.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve items: %w", err)
	}

	if len(items) == 0 {
		return nil, nil
	}

	// Find the most recent item
	var mostRecentItem *Item
	for _, item := range items {
		if mostRecentItem == nil || item.CreatedAt.After(mostRecentItem.CreatedAt) ||
			(item.CreatedAt.Equal(mostRecentItem.CreatedAt) && item.ID > mostRecentItem.ID) {
			mostRecentItem = item
		}
	}

	if mostRecentItem == nil {
		return nil, nil
	}

	// Remove the item from store's items
	s.store.mu.Lock()
	storeItems := s.store.items[s.ID]
	// Find and remove the item
	for i, item := range storeItems {
		if item.ID == mostRecentItem.ID {
			s.store.items[s.ID] = append(storeItems[:i], storeItems[i+1:]...)
			break
		}
	}
	// Also update the session's Items slice
	s.Items = s.store.items[s.ID]
	s.store.mu.Unlock()

	// Return the TResponseInputItem
	return mostRecentItem.ResponseItem.TResponseInputItem, nil
}

// ClearSession clears all items for this session.
func (s *InMemorySession) ClearSession(ctx context.Context) error {
	if s.store == nil {
		return fmt.Errorf("store connection not available")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear items from store
	s.store.mu.Lock()
	s.store.items[s.ID] = []*Item{}
	s.Items = []*Item{}
	s.store.mu.Unlock()

	return nil
}

// Helper function to get tool call id from tool call input
func getToolCallIdFromInput(embeddedItem ResponseItemData) (string, bool) {
	item := embeddedItem.TResponseInputItem

	switch *item.GetType() {
	case string(constant.ValueOf[constant.FunctionCall]()):
		return item.OfFunctionCall.CallID, true
	case string(constant.ValueOf[constant.LocalShellCall]()):
		return item.OfLocalShellCall.CallID, true
	case string(constant.ValueOf[constant.CustomToolCall]()):
		return item.OfCustomToolCall.CallID, true
	default:
		return "", false
	}
}

// Helper function to get tool call id from tool call output
func getToolCallIdFromOutput(embeddedItem ResponseItemData) (string, bool) {
	item := embeddedItem.TResponseInputItem

	switch *item.GetType() {
	case string(constant.ValueOf[constant.FunctionCallOutput]()):
		return item.OfFunctionCallOutput.CallID, true
	case string(constant.ValueOf[constant.ComputerCallOutput]()):
		return item.OfComputerCallOutput.CallID, true
	case string(constant.ValueOf[constant.LocalShellCallOutput]()):
		return item.OfLocalShellCallOutput.ID, true
	case string(constant.ValueOf[constant.CustomToolCallOutput]()):
		return item.OfCustomToolCallOutput.CallID, true
	default:
		return "", false
	}
}
