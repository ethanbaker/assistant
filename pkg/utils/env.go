package utils

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// LoadEnv loads environment variables from multiple .env files
// Returns a map of environment variables, with later files taking precedence
func LoadEnv(files ...string) map[string]string {
	config := make(map[string]string)

	// Load each file in order
	for _, file := range files {
		if _, err := os.Stat(file); err == nil {
			if err := godotenv.Load(file); err != nil {
				log.Printf("[UTILS]: Warning, could not load %s: %v", file, err)
			}
		}
	}

	// Read all environment variables into map
	for _, env := range os.Environ() {
		key, value := splitEnv(env)
		if key != "" {
			config[key] = value
		}
	}

	return config
}

// splitEnv splits an environment variable string into key and value
func splitEnv(env string) (string, string) {
	for i := 0; i < len(env); i++ {
		if env[i] == '=' {
			return env[:i], env[i+1:]
		}
	}
	return "", ""
}

// GetEnvWithDefault returns an environment variable value or a default if not set
func GetEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
