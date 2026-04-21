package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ethanbaker/assistant/pkg/utils"
	"github.com/go-sql-driver/mysql"
)

func main() {
	// Find env file
	envFile := ".env"
	if os.Getenv("ENV_FILE") != "" {
		envFile = os.Getenv("ENV_FILE")
	}

	// Load global config
	cfg := utils.NewConfigFromEnv(envFile)

	// Construct mysql DSN
	mysqlConfig := mysql.Config{
		User:      cfg.Get("MYSQL_USERNAME"),
		Passwd:    cfg.Get("MYSQL_ROOT_PASSWORD"),
		Net:       "tcp",
		Addr:      fmt.Sprintf("%s:%s", cfg.Get("MYSQL_HOST"), cfg.Get("MYSQL_PORT")),
		DBName:    cfg.Get("MYSQL_DATABASE"),
		ParseTime: true,
	}

	// Create SQL store
	store, err := NewSqlStore(mysqlConfig.FormatDSN())
	if err != nil {
		log.Fatalf("[DISCORD]: failed to create SQL store: %v", err)
	}

	// Wait for interrupt signal to gracefully shut down the bot
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Println("[DISCORD]: Starting bot...")

	// Create and start the bot
	bot, err := NewBot(cfg, store)
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

	log.Println("[DISCORD]: Bot stopped gracefully")
}
