package main

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
)

// Config holds runtime configuration for the bot and integration
type Config struct {
	// Discord bot token
	Token string `json:"token"`

	// Base URL for the AI assistant backend
	BackendBaseURL string `json:"backend_base_url"`

	// Channel where the bot will respond to plain messages
	BotChannelID string `json:"bot_channel_id"`

	// Parent channel where conversation threads will be created
	ConversationParentChannelID string `json:"conversation_parent_channel_id"`

	// Optional guild to constrain slash command registration and operations
	GuildID string `json:"guild_id"`
}

// activeConfig is the global reference used by the getter/setter helpers
var activeConfig *Config = &Config{}

// LoadConfig is a placeholder for persistence. User will implement
func LoadConfig(cfg *Config) error { return nil }

// SaveConfig is a placeholder for persistence. User will implement
func SaveConfig(cfg *Config) error { return nil }

// Get reads an environment variable and parses it into T. If the env var is
// not set or parsing fails, it returns the zero value of T
func Get[T any](key string) T {
	var zero T
	tk := tidyKey(key)

	if v, ok := os.LookupEnv(tk); ok {
		if out, ok2 := parseAs[T](v); ok2 {
			return out
		}
	}

	return zero
}

// GetWithDefault behaves like Get but falls back to the provided default if
// the env var is not set or parsing fails
func GetWithDefault[T any](key string, def T) T {
	ck := tidyKey(key)

	if v, ok := os.LookupEnv(ck); ok {
		if out, ok2 := parseAs[T](v); ok2 {
			return out
		}
	}

	return def
}

// Helper method to sanitize keys
func tidyKey(key string) string {
	k := strings.TrimSpace(key)
	k = strings.ReplaceAll(k, "-", "_")
	k = strings.ToUpper(k)
	return k
}

// Helper method to parse generic types
func parseAs[T any](s string) (T, bool) {
	var zero T

	switch any(zero).(type) {
	case string:
		return any(s).(T), true
	case int:
		iv, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return zero, false
		}
		return any(iv).(T), true
	case int64:
		iv, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		if err != nil {
			return zero, false
		}
		return any(iv).(T), true
	case float64:
		fv, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		if err != nil {
			return zero, false
		}
		return any(fv).(T), true
	case bool:
		bv, err := strconv.ParseBool(strings.TrimSpace(s))
		if err != nil {
			return zero, false
		}
		return any(bv).(T), true
	default:
		var out T
		if err := json.Unmarshal([]byte(s), &out); err != nil {
			return zero, false
		}
		return out, true
	}
}
