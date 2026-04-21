package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethanbaker/assistant/pkg/config"
	"github.com/ethanbaker/assistant/pkg/logger"
	"github.com/ethanbaker/assistant/pkg/sdk"
	"github.com/go-sql-driver/mysql"
)

func main() {
	// Load environment file (.env by default, overridden by ENV_FILE)
	envFile := ".env"
	if v := os.Getenv("ENV_FILE"); v != "" {
		envFile = v
	}
	config.Load(envFile)

	// Load outreach job definitions from YAML config file
	jobs, err := loadOutreachJobs()
	if err != nil {
		logger.Fatalf("failed to load outreach jobs: %v", err)
	}

	// Build MySQL DSN and create the SQL repository
	dsn := mysqlDSN()
	repo, err := NewSqlRepository(dsn)
	if err != nil {
		logger.Fatalf("failed to create SQL repository: %v", err)
	}

	clientKeyRepo, err := NewSqlOutreachClientRepository(dsn)
	if err != nil {
		logger.Fatalf("failed to create outreach client repository: %v", err)
	}

	// Create context to gracefully manage shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create new SDK instance
	sdkClient := sdk.NewClient(config.MustGetenv("BACKEND_BASE_URL"), config.MustGetenv("BACKEND_API_KEY"))

	// Create new discord bot
	bot, err := NewBot(BotConfig{
		SDKClient:              sdkClient,
		ConversationRepository: repo,
		Token:                  config.MustGetenv("DISCORD_TOKEN"),
		BotChannelID:           config.MustGetenv("BOT_CHANNEL_ID"),
		ThreadChannelID:        config.MustGetenv("THREAD_CHANNEL_ID"),
		UserID:                 config.MustGetenv("USER_ID"),
		GuildID:                config.GetenvValue("GUILD_ID"),
	})

	// Create outreach manager
	outreachManager, err := NewOutreachManager(OutreachManagerConfig{
		Jobs:          jobs,
		SDKClient:     sdkClient,
		ClientName:    config.MustGetenv("OUTREACH_CLIENT_NAME"),
		Host:          config.MustGetenv("OUTREACH_HOST"),
		Port:          config.GetenvWithDefault("OUTREACH_PORT", "9000"),
		ClientKeyRepo: clientKeyRepo,
	})
	if err != nil {
		logger.Fatalf("failed to create outreach manager: %v", err)
	}

	// Add handler for the discord response
	if err := outreachManager.addHandler("/api/outreach-message", bot.onOutreachMessage); err != nil {
		logger.Fatalf("failed to add outreach handler from discord bot: %v", err)
	}

	// Unsubscribe from existing jobs
	logger.Info("unsubscribing from existing outreach jobs")
	_ = outreachManager.unsubscribe()

	// Subscribe to jobs
	logger.Info("resubscribing to outreach jobs")
	if err := outreachManager.subscribe(); err != nil {
		logger.Fatalf("failed to subscribe to outreach events: %v", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		logger.Warn("quit signal received")
		cancel()
	}()

	// Start the outreach api
	logger.Info("starting outreach api...")
	outreachManager.start()
	logger.Info("outreach is running")

	// Start the bot
	logger.Info("starting bot...")
	if err := bot.start(); err != nil {
		logger.Fatalf("failed to start bot: %v", err)
	}
	logger.Info("bot is running")

	logger.Info("Press Ctrl+C to exit")
	<-ctx.Done()

	// Create a fresh context for graceful shutdown (original is already cancelled)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	// Handle cleanup
	logger.Info("shutting down outreach...")
	if err := outreachManager.stop(shutdownCtx); err != nil {
		logger.Errorf("error during outreach shutdown: %v", err)
	}

	logger.Info("shutting down bot...")
	if err := bot.stop(); err != nil {
		logger.Errorf("error during bot shutdown: %v", err)
	}

	logger.Info("stopped gracefully")
}

// mysqlDSN builds a MySQL DSN from environment variables.
func mysqlDSN() string {
	cfg := mysql.Config{
		User:      config.GetenvValue("MYSQL_USERNAME"),
		Passwd:    config.GetenvValue("MYSQL_ROOT_PASSWORD"),
		Net:       "tcp",
		Addr:      fmt.Sprintf("%s:%s", config.GetenvValue("MYSQL_HOST"), config.GetenvValue("MYSQL_PORT")),
		DBName:    config.GetenvValue("MYSQL_DATABASE"),
		ParseTime: true,
	}
	return cfg.FormatDSN()
}
