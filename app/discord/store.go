package main

import (
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// DiscordSession represents a discord session mapping in the database
// Maps DiscordID -> SessionUUID
type DiscordSession struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	DiscordID string `gorm:"uniqueIndex;not null;size:191"` // Discord channel/thread ID
	SessionID string `gorm:"not null;size:191"`             // Session UUID
}

// TableName sets the table name for GORM
func (DiscordSession) TableName() string {
	return "discord_sessions"
}

// InMemoryStore is a simple in-memory store for managing conversations and their associated session UUIDs
type InMemoryStore struct {
	channel map[string]string // unique mapping (channelID, threadID, etc) -> session UUID
}

// NewInMemoryStore initializes a new InMemoryStore
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{channel: make(map[string]string)}
}

// Get retrieves the session UUID for a given key (channel or thread ID)
func (s *InMemoryStore) Get(key string) (string, bool) {
	v, ok := s.channel[key]
	return v, ok
}

// Set associates a session UUID with a key (channel or thread ID)
func (s *InMemoryStore) Set(key, uuid string) {
	s.channel[key] = uuid
}

// Delete removes the session UUID associated with a key (channel or thread ID)
func (s *InMemoryStore) Delete(key string) {
	delete(s.channel, key)
}

// SqlStore is a SQL implementation of the store interface
type SqlStore struct {
	db *gorm.DB
}

// NewSqlStore initializes a new SqlStore
func NewSqlStore(dsn string) (*SqlStore, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return &SqlStore{db: db}, db.AutoMigrate(&DiscordSession{})
}

// Get retrieves the session UUID for a given key (channel or thread ID)
func (s *SqlStore) Get(key string) (string, bool) {
	var session DiscordSession
	result := s.db.Where("discord_id = ?", key).First(&session)
	if result.Error != nil {
		log.Printf("[SQL-STORE]: error retrieving session for key %s: %v", key, result.Error)
	}
	return session.SessionID, result.Error == nil
}

// Set associates a session UUID with a key (channel or thread ID)
func (s *SqlStore) Set(key, uuid string) {
	session := DiscordSession{DiscordID: key, SessionID: uuid}
	s.db.Save(&session)
}

// Delete removes the session UUID associated with a key (channel or thread ID)
func (s *SqlStore) Delete(key string) {
	s.db.Where("discord_id = ?", key).Delete(&DiscordSession{})
}
