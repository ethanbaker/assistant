package main

import (
	"os"

	"github.com/ethanbaker/assistant/internal/api"
	"github.com/ethanbaker/assistant/pkg/utils"
)

// Start the API server
func main() {
	// Find env file
	envFile := ".env"
	if os.Getenv("ENV_FILE") != "" {
		envFile = os.Getenv("ENV_FILE")
	}

	// Load global config
	cfg := utils.NewConfigFromEnv(envFile)

	// Start
	api.Start(cfg)
}
