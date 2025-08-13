package utils

import (
	"maps"
	"os"
	"strconv"
	"sync"
)

// Config provides a thread-safe configuration management system
// that handles environment variables with defaults, type conversion, and modification
type Config struct {
	mu     sync.RWMutex
	values map[string]string
}

// NewConfig creates a new Config instance with the provided key-value pairs
func NewConfig(values map[string]string) *Config {
	config := &Config{
		values: make(map[string]string),
	}

	maps.Copy(config.values, values)

	return config
}

// NewConfigFromEnv creates a new Config instance by loading environment variables
// from the specified .env files (similar to LoadEnv)
func NewConfigFromEnv(files ...string) *Config {
	envMap := LoadEnv(files...)
	return NewConfig(envMap)
}

// Get retrieves a configuration value by key
// Returns empty string if key doesn't exist
func (c *Config) Get(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.values[key]
}

// GetWithDefault retrieves a configuration value by key with a fallback default
func (c *Config) GetWithDefault(key, defaultValue string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if value, exists := c.values[key]; exists && value != "" {
		return value
	}
	return defaultValue
}

// GetBool retrieves a configuration value as a boolean
// Returns false if key doesn't exist or cannot be parsed as boolean
func (c *Config) GetBool(key string) bool {
	value := c.Get(key)
	if value == "" {
		return false
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		// Handle common boolean representations
		switch value {
		case "1", "yes", "on", "enabled":
			return true
		case "0", "no", "off", "disabled":
			return false
		default:
			return false
		}
	}
	return parsed
}

// GetBoolWithDefault retrieves a configuration value as a boolean with a fallback default
func (c *Config) GetBoolWithDefault(key string, defaultValue bool) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if _, exists := c.values[key]; !exists {
		return defaultValue
	}

	return c.GetBool(key)
}

// GetInt retrieves a configuration value as an integer
// Returns 0 if key doesn't exist or cannot be parsed as integer
func (c *Config) GetInt(key string) int {
	value := c.Get(key)
	if value == "" {
		return 0
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}

// GetIntWithDefault retrieves a configuration value as an integer with a fallback default
func (c *Config) GetIntWithDefault(key string, defaultValue int) int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if _, exists := c.values[key]; !exists {
		return defaultValue
	}

	return c.GetInt(key)
}

// Set modifies a configuration value
func (c *Config) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = value
}

// SetBool modifies a configuration value as a boolean
func (c *Config) SetBool(key string, value bool) {
	c.Set(key, strconv.FormatBool(value))
}

// SetInt modifies a configuration value as an integer
func (c *Config) SetInt(key string, value int) {
	c.Set(key, strconv.Itoa(value))
}

// Delete removes a configuration key
func (c *Config) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.values, key)
}

// Has checks if a configuration key exists
func (c *Config) Has(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, exists := c.values[key]
	return exists
}

// Keys returns all configuration keys
func (c *Config) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.values))
	for k := range c.values {
		keys = append(keys, k)
	}
	return keys
}

// ToMap returns a copy of all configuration values as a map
// This is useful for backwards compatibility with existing code that expects map[string]string
func (c *Config) ToMap() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]string, len(c.values))
	maps.Copy(result, c.values)
	return result
}

// Merge combines values from another config, with the other config taking precedence
func (c *Config) Merge(other *Config) {
	if other == nil {
		return
	}

	c.mu.Lock()
	other.mu.RLock()
	defer c.mu.Unlock()
	defer other.mu.RUnlock()

	maps.Copy(c.values, other.values)
}

// SyncWithEnv updates the config with current environment variables
// This allows runtime updates to environment variables to be reflected
func (c *Config) SyncWithEnv() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update existing keys with current env values
	for key := range c.values {
		if envValue := os.Getenv(key); envValue != "" {
			c.values[key] = envValue
		}
	}
}

// Clone creates a deep copy of the config
func (c *Config) Clone() *Config {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return NewConfig(c.values)
}
