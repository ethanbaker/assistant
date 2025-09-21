package outreach

import (
	"time"

	"gorm.io/gorm"
)

// ImplementationModel represents the database model for outreach implementations
type ImplementationModel struct {
	ID        uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;index"`

	ClientID     string `json:"client_id" gorm:"column:client_id;unique;not null;size:255"`
	CallbackURL  string `json:"callback_url" gorm:"column:callback_url;not null;size:500"`
	ClientSecret string `json:"client_secret" gorm:"column:client_secret;size:255"`
	Active       bool   `json:"active" gorm:"column:active;default:true"`
}

// TableName sets the table name for GORM
func (ImplementationModel) TableName() string {
	return "outreach_implementations"
}
