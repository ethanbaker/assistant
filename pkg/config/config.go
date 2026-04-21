// config.go - contains app-level os env wrapper
package config

import (
	"log"

	"github.com/joho/godotenv"
)

var env map[string]string

// Load initializes an environment from an env file
func Load(path string) {
	var err error

	// Load environment variables
	if err = godotenv.Load(path); err != nil {
		log.Printf("Failed to load env")
		panic(err)
	}

	// Read environment variables into the global map
	env, err = godotenv.Read(path)
	if err != nil {
		log.Printf("Failed to read env")
		panic(err)
	}

}

// Getenv returns the value and existence status of the environment variable with the given key
func Getenv(key string) (string, bool) {
	val, ok := env[key]
	if !ok {
		return "", false
	}
	return val, true
}

// MustGetenv returns the value of an environment variable or panics if no value is set
func MustGetenv(key string) string {
	val, ok := env[key]
	if !ok {
		panic("environment variable '" + key + "' must be set and was not found")
	}
	return val
}

// GetenvValue returns the value of the environment variable with the given key. If no such key exists, an empty string is returned
func GetenvValue(key string) string {
	val, ok := env[key]
	if !ok {
		return ""
	}

	return val
}

// GetEnvWithDefault returns the value of the environment variable with the given key. If no such key exists, the default value is returned
func GetenvWithDefault(key, defaultValue string) string {
	if _, ok := env[key]; !ok {
		return defaultValue
	}

	return env[key]
}

// KeyExists returns true if the environment variable with the given key exists, false otherwise
func KeyExists(key string) bool {
	_, ok := env[key]
	return ok
}
