package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ethanbaker/assistant/pkg/utils"
)

func main() {
	// Find env file
	envFile := ".env"
	if os.Getenv("ENV_FILE") != "" {
		envFile = os.Getenv("ENV_FILE")
	}

	// Load global config
	cfg := utils.NewConfigFromEnv(envFile)

	// Wait for interrupt signal to gracefully shut down the app
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Println("[LISTEN]: Starting voice assistant...")

	// Create and start the voice assistant
	assistant, err := NewVoiceAssistant(cfg)
	if err != nil {
		log.Fatalf("[LISTEN]: failed to create voice assistant: %v", err)
	}

	if err := assistant.Start(ctx); err != nil {
		log.Fatalf("[LISTEN]: failed to start voice assistant: %v", err)
	}

	// Wait for shutdown signal
	log.Println("[LISTEN]: Voice assistant is running. Press Ctrl+C to exit.")
	<-ctx.Done()

	// Cleanly stop the assistant
	if err := assistant.Stop(); err != nil {
		log.Printf("error during voice assistant shutdown: %v", err)
	}

	log.Println("[LISTEN]: Voice assistant stopped")
}
