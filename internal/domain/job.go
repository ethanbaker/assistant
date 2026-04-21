package domain

import (
	"encoding/json"

	"gorm.io/gorm"
)

// Job defines a scheduled outreach task.
type Job struct {
	*gorm.Model
	Active     bool
	Name       string
	Schedule   json.RawMessage `gorm:"type:json"`
	Handler    string
	Parameters json.RawMessage `gorm:"type:json"`

	Subscriptions []JobSubscription `gorm:"foreignKey:JobId"`
}
