package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ethanbaker/assistant/pkg/sdk"
	"github.com/ethanbaker/assistant/pkg/utils"
)

const NO_CONTENT = "(no content)"
const THREAD_ARCHIVE = 1440 // 24 hours

// Bot represents the Discord bot instance
type Bot struct {
	config *utils.Config      // Configuration struct
	dg     *discordgo.Session // Discord session
	api    *sdk.Client        // Backend API client

	conversations *ConversationStore // In-memory store for conversations

	// Important configuration values
	botChannelID           string // Channel ID where the bot listens for messages
	botChannelContextLimit int    // Limit for bot channel messages

	threadChannelID           string // Channel ID where conversation threads are created
	threadChannelContextLimit int    // Limit for thread context messages
	guildID                   string // Guild ID for slash commands (empty for global)
}

// ConversationStore is a simple in-memory store for managing conversations and their associated session UUIDs
// Keying strategy:
// - For channel free-chat: key is channelID
// - For thread conversation: key is threadID
// '/ask' commands are not stored here, as they are one-off and don't require persistence
type ConversationStore struct {
	channel map[string]string // unique mapping (channelID, threadID, etc) -> session UUID
}

// NewConversationStore initializes a new ConversationStore
func NewConversationStore() *ConversationStore {
	return &ConversationStore{channel: make(map[string]string)}
}

// Get retrieves the session UUID for a given key (channel or thread ID)
func (s *ConversationStore) Get(key string) (string, bool) {
	v, ok := s.channel[key]
	return v, ok
}

// Set associates a session UUID with a key (channel or thread ID)
func (s *ConversationStore) Set(key, uuid string) {
	s.channel[key] = uuid
}

// Delete removes the session UUID associated with a key (channel or thread ID)
func (s *ConversationStore) Delete(key string) {
	delete(s.channel, key)
}

// Create a new Discord bot instance
func NewBot(cfg *utils.Config) (*Bot, error) {
	// Get discord token
	token := cfg.Get("DISCORD_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("DISCORD_TOKEN not set in config or environment")
	}

	// Get important configuration values
	botChannelID := cfg.Get("BOT_CHANNEL_ID")
	if botChannelID == "" {
		return nil, fmt.Errorf("BOT_CHANNEL_ID not set in config or environment")
	}

	botChannelContextLimit := cfg.GetIntWithDefault("BOT_CHANNEL_CONTEXT_LIMIT", 15)
	if botChannelContextLimit <= 0 {
		return nil, fmt.Errorf("BOT_CHANNEL_CONTEXT_LIMIT must be a positive integer")
	}

	threadChannelID := cfg.Get("THREAD_CHANNEL_ID")
	if threadChannelID == "" {
		return nil, fmt.Errorf("THREAD_CHANNEL_ID not set in config or environment")
	}

	threadChannelContextLimit := cfg.GetIntWithDefault("THREAD_CHANNEL_CONTEXT_LIMIT", 15)
	if threadChannelContextLimit <= 0 {
		return nil, fmt.Errorf("THREAD_CHANNEL_CONTEXT_LIMIT must be a positive integer")
	}

	guildID := cfg.Get("GUILD_ID") // empty = global commands
	if guildID == "" {
		log.Println("GUILD_ID not set, using global commands")
	}

	// Get base URL and api key
	baseURL := cfg.Get("BACKEND_BASE_URL")
	if baseURL == "" {
		return nil, fmt.Errorf("BACKEND_BASE_URL not set in config or environment")
	}

	apiKey := cfg.Get("BACKEND_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("BACKEND_API_KEY not set in config or environment")
	}

	// Create a new Discord session
	dg, err := discordgo.New("Bot " + strings.TrimPrefix(token, "Bot "))
	if err != nil {
		return nil, err
	}

	// Create the bot instance
	b := &Bot{
		config:                    cfg,
		dg:                        dg,
		api:                       sdk.NewClient(baseURL, apiKey),
		conversations:             NewConversationStore(),
		botChannelID:              botChannelID,
		botChannelContextLimit:    botChannelContextLimit,
		threadChannelID:           threadChannelID,
		guildID:                   guildID,
		threadChannelContextLimit: threadChannelContextLimit,
	}

	// Intents
	dg.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsMessageContent |
		discordgo.IntentsGuildMessageReactions |
		discordgo.IntentsDirectMessages

	// Handlers
	dg.AddHandler(b.onReady)
	dg.AddHandler(b.onMessageCreate)
	dg.AddHandler(b.onInteractionCreate)

	return b, nil
}

// Start the bot and connect to Discord
func (b *Bot) Start() error {
	if err := b.dg.Open(); err != nil {
		return err
	}

	// Register slash commands
	return b.registerCommands()
}

// Stop the bot and clean up resources
func (b *Bot) Stop() error {
	_ = b.unregisterCommands()
	return b.dg.Close()
}

// onReady is called when the bot is ready
func (b *Bot) onReady(s *discordgo.Session, r *discordgo.Ready) {
	log.Printf("[DISCORD]: Logged in as: %s#%s", r.User.Username, r.User.Discriminator)
}

// onMessageCreate handles incoming messages
func (b *Bot) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot itself
	if m.Author == nil || m.Author.ID == s.State.User.ID {
		return
	}

	// Ignore empty messages
	content := strings.TrimSpace(m.Content)
	if content == "" {
		return
	}

	// If it's the configured bot channel, handle
	if m.ChannelID == b.botChannelID {
		go b.handleMessageInChannel(m.ChannelID, m.Author, content, m.Message)
		return
	}

	// If the channel has a bound conversation, handle.
	if _, ok := b.conversations.Get(m.ChannelID); ok {
		go b.handleMessageInChannel(m.ChannelID, m.Author, content, m.Message)
	}
}

// handleMessageInChannel processes messages in the bot channel or bound conversation channels
func (b *Bot) handleMessageInChannel(channelID string, user *discordgo.User, content string, msg *discordgo.Message) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Ensure a conversation exists per channel
	conversationID, ok := b.conversations.Get(channelID)
	if !ok {
		// Create new session if not found
		sess, err := b.api.CreateSession(ctx, &sdk.CreateSessionRequest{
			UserID: user.ID,
		})
		if err != nil {
			errorReply(b.dg, channelID, "Failed to create session", err)
			return
		}

		// Bind the session to the channel
		conversationID = sess.ID
		b.conversations.Set(channelID, conversationID)
	}

	// Add the message to the session
	resp, err := b.api.SendMessage(ctx, conversationID, &sdk.PostMessageRequest{
		Content: decorateDiscordContext(user, content),
	})
	if err != nil {
		errorReply(b.dg, channelID, "Failed to send message", err)
		return
	}

	// If the response is empty, just return filler
	output := strings.TrimSpace(resp.FinalOutput)
	if output != "" {
		reply(b.dg, channelID, output)
	}
}
