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

	// TODO: load conversation mapping

	// Wait for interrupt signal to gracefully shut down the bot
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Println("[DISCORD]: Starting bot...")

	// Create and start the bot
	bot, err := NewBot(cfg)
	if err != nil {
		log.Fatalf("[DISCORD]: failed to create bot: %v", err)
	}

	if err := bot.Start(); err != nil {
		log.Fatalf("[DISCORD]: failed to start bot: %v", err)
	}

	// Wait for shutdown signal
	log.Println("[DISCORD]: Bot is running. Press Ctrl+C to exit.")
	<-ctx.Done()

	// Cleanly stop the bot
	if err := bot.Stop(); err != nil {
		log.Printf("error during bot shutdown: %v", err)
	}

	// Save config on clean shutdown
	// TODO: save conversation mapping

	log.Println("[DISCORD]: Bot stopped gracefully")
}
