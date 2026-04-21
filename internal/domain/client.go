package domain

import "gorm.io/gorm"

// Client is an outbound webhook consumer.
type Client struct {
	*gorm.Model
	Name       string
	ApiKey     string
	WebhookUrl string

	Subscriptions []JobSubscription `gorm:"foreignKey:ClientId"`
}
