package main

import (
	"github.com/ethanbaker/assistant/pkg/logger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ConversationRepository manages the mapping from Discord channel/thread IDs to session UUIDs.
//
// Keying strategy:
//   - Bot channel free-chat: key is channelID
//   - Thread conversation: key is threadID
//   - /ask commands are not stored (ephemeral, one-off)
type ConversationRepository interface {
	Get(key string) (string, bool)
	Set(key, uuid string)
	Delete(key string)
}

// DiscordSession maps a Discord channel/thread ID to an agent session UUID.
type DiscordSession struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	DiscordID string `gorm:"uniqueIndex;not null;size:191"`
	SessionID string `gorm:"not null;size:191"`
}

func (DiscordSession) TableName() string {
	return "discord_sessions"
}

// InMemoryRepository is an in-memory ConversationRepository, useful for testing.
type InMemoryRepository struct {
	data map[string]string
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{data: make(map[string]string)}
}

func (r *InMemoryRepository) Get(key string) (string, bool) {
	v, ok := r.data[key]
	return v, ok
}

func (r *InMemoryRepository) Set(key, uuid string) {
	r.data[key] = uuid
}

func (r *InMemoryRepository) Delete(key string) {
	delete(r.data, key)
}

// SqlRepository is a MySQL-backed ConversationRepository using GORM.
type SqlRepository struct {
	db *gorm.DB
}

func NewSqlRepository(dsn string) (*SqlRepository, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return &SqlRepository{db: db}, db.AutoMigrate(&DiscordSession{})
}

func (r *SqlRepository) Get(key string) (string, bool) {
	var session DiscordSession
	result := r.db.Where("discord_id = ?", key).First(&session)
	if result.Error != nil {
		logger.Errorf("error retrieving session for key %s: %v", key, result.Error)
		return "", false
	}
	return session.SessionID, true
}

func (r *SqlRepository) Set(key, uuid string) {
	session := DiscordSession{DiscordID: key, SessionID: uuid}
	r.db.Save(&session)
}

func (r *SqlRepository) Delete(key string) {
	r.db.Where("discord_id = ?", key).Delete(&DiscordSession{})
}

// OutreachClient persists the outreach client API key across restarts.
type OutreachClient struct {
	ID     uint   `gorm:"primaryKey;autoIncrement"`
	APIKey string `gorm:"not null;size:191"`
}

func (OutreachClient) TableName() string {
	return "discord_outreach_client"
}

// OutreachClientRepository manages the stored outreach client key.
type OutreachClientRepository interface {
	Get() (string, bool)
	Save(key string) error
}

// SqlOutreachClientRepository is a MySQL-backed OutreachClientRepository using GORM.
type SqlOutreachClientRepository struct {
	db *gorm.DB
}

func NewSqlOutreachClientRepository(dsn string) (*SqlOutreachClientRepository, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return &SqlOutreachClientRepository{db: db}, db.AutoMigrate(&OutreachClient{})
}

func (r *SqlOutreachClientRepository) Get() (string, bool) {
	var client OutreachClient
	if err := r.db.First(&client).Error; err != nil {
		return "", false
	}
	return client.APIKey, true
}

func (r *SqlOutreachClientRepository) Save(key string) error {
	var client OutreachClient
	r.db.First(&client)
	client.APIKey = key
	return r.db.Save(&client).Error
}
