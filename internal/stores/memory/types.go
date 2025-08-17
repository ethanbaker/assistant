package memory

import (
	"time"

	"gorm.io/gorm"
)

// KeyFact represents a stored fact in the memory system
type KeyFact struct {
	ID        uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"column:deleted_at;index"`

	Key   string `json:"key" gorm:"column:fact_key;unique;not null;size:255"`
	Value string `json:"value" gorm:"type:text"`
}

// TableName sets the table name for GORM
func (KeyFact) TableName() string {
	return "key_facts"
}

// NewKeyFact creates a new key fact with auto-incrementing ID
func NewKeyFact(key, value string) *KeyFact {
	return &KeyFact{
		Key:   key,
		Value: value,
	}
}
