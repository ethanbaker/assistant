package session

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nlpodyssey/openai-agents-go/memory"
	"github.com/openai/openai-go/v2/packages/param"
	"github.com/openai/openai-go/v2/responses"
	"gorm.io/gorm"
)

// ResponseItemData is a wrapper type that implements database serialization
type ResponseItemData struct {
	*memory.TResponseInputItem
}

// Value implements the driver.Valuer interface for database storage
func (r ResponseItemData) Value() (driver.Value, error) {
	if r.TResponseInputItem == nil {
		return nil, nil
	}
	return json.Marshal(r.TResponseInputItem)
}

// Scan implements the sql.Scanner interface for database retrieval
func (r *ResponseItemData) Scan(value any) error {
	if value == nil {
		r.TResponseInputItem = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into ResponseItemData", value)
	}

	// Unmarshal the JSON bytes into the TResponseInputItem
	item := &memory.TResponseInputItem{}
	err := json.Unmarshal(bytes, item)
	if err != nil {
		return fmt.Errorf("failed to unmarshal ResponseItemData: %w", err)
	}

	// Fix incorrect output messages unmarshaling (from openai-agents-go)
	if msg := item.OfMessage; !param.IsOmitted(msg) {
		if msg.Content.OfInputItemContentList == nil && msg.Content.OfString == (param.Opt[string]{}) {
			var outMsg responses.ResponseOutputMessageParam
			err = json.Unmarshal(bytes, &outMsg)
			if err == nil && len(outMsg.Content) > 0 && !param.IsOmitted(outMsg.Content[0].OfOutputText) && outMsg.Content[0].OfOutputText.Text != "" {
				item = &memory.TResponseInputItem{
					OfOutputMessage: &outMsg,
				}
			}
		}
	}

	// Set the corrected item and return
	r.TResponseInputItem = item
	return nil
}

// Item represents a single item in a session. It can include messages or agent actions
type Item struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"column:deleted_at;index"`

	ResponseItem ResponseItemData `json:"data" gorm:"column:data;type:text;not null"` // OpenAI SDK type for response input items

	// Session information
	SessionID uuid.UUID `json:"session_id" gorm:"type:char(36);not null;index"`
}

// NewItem creates a new item
func NewItem(sessionID uuid.UUID, responseItem *memory.TResponseInputItem) *Item {
	return &Item{
		SessionID: sessionID,
		ResponseItem: ResponseItemData{
			TResponseInputItem: responseItem,
		},
	}
}
