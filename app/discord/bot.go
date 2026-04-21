package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ethanbaker/assistant/pkg/logger"
	"github.com/ethanbaker/assistant/pkg/sdk"
	"github.com/gin-gonic/gin"
)

const noContent = "(no content)"
const threadArchiveDuration = 1440 // 24 hours in minutes

// Bot represents the Discord bot instance.
type Bot struct {
	dg  *discordgo.Session
	api *sdk.Client

	conversations ConversationRepository

	// Discord config
	botChannelID    string
	threadChannelID string
	guildID         string
	userID          string
}

type BotConfig struct {
	SDKClient              *sdk.Client
	ConversationRepository ConversationRepository

	Token           string
	BotChannelID    string
	ThreadChannelID string
	UserID          string
	GuildID         string
}

// NewBot creates a new Bot from provided config
func NewBot(cfg BotConfig) (*Bot, error) {
	// Validate config
	if cfg.SDKClient == nil {
		return nil, fmt.Errorf("sdk client is required")
	}
	if cfg.ConversationRepository == nil {
		return nil, fmt.Errorf("conversation repository is required")
	}
	token := strings.TrimSpace(cfg.Token)
	if token == "" {
		return nil, fmt.Errorf("token is required")
	}
	botChannelId := strings.TrimSpace(cfg.BotChannelID)
	if botChannelId == "" {
		return nil, fmt.Errorf("bot channel id is required")
	}
	threadChannelId := strings.TrimSpace(cfg.ThreadChannelID)
	if threadChannelId == "" {
		return nil, fmt.Errorf("thread channel id is required")
	}
	userId := strings.TrimSpace(cfg.UserID)
	if userId == "" {
		return nil, fmt.Errorf("user id is required")
	}
	guildId := strings.TrimSpace(cfg.GuildID)
	if guildId == "" {
		logger.Info("GUILD_ID not set, using global commands")
	}

	// Create new discord session
	dg, err := discordgo.New("Bot " + strings.TrimPrefix(token, "Bot "))
	if err != nil {
		return nil, fmt.Errorf("failed to create discord session: %w", err)
	}

	// Declare intents
	dg.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsMessageContent |
		discordgo.IntentsGuildMessageReactions |
		discordgo.IntentsDirectMessages

	// Create new bot instance
	b := &Bot{
		dg:              dg,
		api:             cfg.SDKClient,
		conversations:   cfg.ConversationRepository,
		botChannelID:    botChannelId,
		threadChannelID: threadChannelId,
		guildID:         guildId,
		userID:          userId,
	}

	// Create handlers
	dg.AddHandler(b.onReady)
	dg.AddHandler(b.onMessageCreate)
	dg.AddHandler(b.onInteractionCreate)

	return b, nil
}

// start opens the Discord connection and registers slash commands.
func (b *Bot) start() error {
	if err := b.dg.Open(); err != nil {
		return err
	}
	return b.registerCommands()
}

// stop cleanly shuts down the bot.
func (b *Bot) stop() error {
	_ = b.unregisterCommands()
	return b.dg.Close()
}

func (b *Bot) onReady(s *discordgo.Session, r *discordgo.Ready) {
	logger.Infof("logged in as %s#%s", r.User.Username, r.User.Discriminator)
}

func (b *Bot) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages the bot created
	if m.Author == nil || m.Author.ID == s.State.User.ID {
		return
	}

	// Ignore empty messages
	content := strings.TrimSpace(m.Content)
	if content == "" {
		return
	}

	// Free-chat in the configured bot channel
	if m.ChannelID == b.botChannelID {
		go b.handleMessageInChannel(m.ChannelID, m.Author, content)
		return
	}

	// Any channel with a bound conversation (e.g. a thread)
	if _, ok := b.conversations.Get(m.ChannelID); ok {
		go b.handleMessageInChannel(m.ChannelID, m.Author, content)
	}
}

// handleMessageInChannel routes a message through the agent API and replies.
func (b *Bot) handleMessageInChannel(channelID string, user *discordgo.User, content string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	sessionID, ok := b.conversations.Get(channelID)
	if !ok {
		sess, err := b.api.CreateSession(ctx, &sdk.CreateSessionRequest{
			UserID: user.ID,
		})
		if err != nil {
			errorReply(b.dg, channelID, "Failed to create session", err)
			return
		}

		sessionID = sess.ID
		b.conversations.Set(channelID, sessionID)
	}

	resp, err := b.api.SendMessage(ctx, sessionID, &sdk.PostMessageRequest{
		Content: decorateDiscordContext(user, content),
	})
	if err != nil {
		errorReply(b.dg, channelID, "Failed to send message", err)
		return
	}

	if output := strings.TrimSpace(resp.FinalOutput); output != "" {
		reply(b.dg, channelID, output)
	}
}

// onOutreachMessage handles incoming webhook calls from the backend outreach system.
func (b *Bot) onOutreachMessage(c *gin.Context, payload sdk.WebhookPayload) error {
	if b.dg == nil {
		return fmt.Errorf("discord session is nil")
	}

	dmChannel, err := b.dg.UserChannelCreate(b.userID)
	if err != nil {
		logger.Errorf("failed to create DM channel with user %s: %v", b.userID, err)
		return fmt.Errorf("failed to create dm channel: %v", err)
	}

	replySanitizeHTML(b.dg, dmChannel.ID, payload.Content)

	logger.Infof("sent outreach message to user %s (id=%d)", b.userID, payload.ID)
	return nil
}
