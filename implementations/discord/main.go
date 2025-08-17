package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Load config (user will implement persistence).
	cfg := DefaultConfig()
	_ = LoadConfig(cfg)

	// Allow env vars to override
	if token := os.Getenv("DISCORD_TOKEN"); token != "" {
		cfg.Token = token
	}
	if base := os.Getenv("BACKEND_BASE_URL"); base != "" {
		cfg.BackendBaseURL = base
	}
	if ch := os.Getenv("BOT_CHANNEL_ID"); ch != "" {
		cfg.BotChannelID = ch
	}
	if th := os.Getenv("CONV_PARENT_CHANNEL_ID"); th != "" {
		cfg.ConversationParentChannelID = th
	}
	if gid := os.Getenv("GUILD_ID"); gid != "" {
		cfg.GuildID = gid
	}

	if cfg.Token == "" {
		log.Fatal("DISCORD_TOKEN not set in config or environment")
	}
	if cfg.BackendBaseURL == "" {
		log.Fatal("BACKEND_BASE_URL not set in config or environment")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	bot, err := NewBot(cfg)
	if err != nil {
		log.Fatalf("failed to create bot: %v", err)
	}

	if err := bot.Start(); err != nil {
		log.Fatalf("failed to start bot: %v", err)
	}
	log.Println("Bot is running. Press Ctrl+C to exit.")

	<-ctx.Done()

	if err := bot.Stop(); err != nil {
		log.Printf("error during bot shutdown: %v", err)
	}

	// Save config on clean shutdown
	if err := SaveConfig(cfg); err != nil {
		log.Printf("failed to save config: %v", err)
	}
}
