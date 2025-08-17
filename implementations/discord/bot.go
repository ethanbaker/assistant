package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	cfg *Config
	dg  *discordgo.Session
	api *BackendClient
	// Map Discord user or channel to session id.
	sessions *SessionStore
}

type SessionStore struct {
	// In-memory map; persistence hooks exposed via Save/LoadConfig for now.
	// Keying strategy:
	// - For channel free-chat: key is channelID
	// - For slash /ask ephemeral: create temp session per interaction
	// - For thread conversation: key is threadID
	channel map[string]string // channelID -> sessionUUID
}

func NewSessionStore() *SessionStore { return &SessionStore{channel: make(map[string]string)} }

func (s *SessionStore) Get(key string) (string, bool) {
	v, ok := s.channel[key]
	return v, ok
}
func (s *SessionStore) Set(key, uuid string) { s.channel[key] = uuid }
func (s *SessionStore) Delete(key string)    { delete(s.channel, key) }

func NewBot(cfg *Config) (*Bot, error) {
	// Accept raw token; discordgo requires just the raw token with Bot prefix handled by lib.
	dg, err := discordgo.New("Bot " + strings.TrimPrefix(cfg.Token, "Bot "))
	if err != nil {
		return nil, err
	}

	b := &Bot{cfg: cfg, dg: dg, api: NewBackendClient(cfg.BackendBaseURL), sessions: NewSessionStore()}

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

func (b *Bot) Start() error {
	if err := b.dg.Open(); err != nil {
		return err
	}
	// Register slash commands
	return b.registerCommands()
}

func (b *Bot) Stop() error {
	_ = b.unregisterCommands()
	return b.dg.Close()
}

func (b *Bot) onReady(s *discordgo.Session, r *discordgo.Ready) {
	log.Printf("Logged in as: %s#%s", r.User.Username, r.User.Discriminator)
}

func (b *Bot) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot itself
	if m.Author == nil || m.Author.ID == s.State.User.ID {
		return
	}

	content := strings.TrimSpace(m.Content)
	if content == "" {
		return
	}

	// If it's the configured bot channel, handle.
	if b.cfg.BotChannelID != "" && m.ChannelID == b.cfg.BotChannelID {
		go b.handleMessageInChannel(m.ChannelID, m.Author, content, m.Message)
		return
	}
	// If the channel (e.g., a thread) has a bound session, handle.
	if _, ok := b.sessions.Get(m.ChannelID); ok {
		go b.handleMessageInChannel(m.ChannelID, m.Author, content, m.Message)
	}
}

func (b *Bot) handleMessageInChannel(channelID string, user *discordgo.User, content string, msg *discordgo.Message) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Ensure a session exists per channel
	sessionID, ok := b.sessions.Get(channelID)
	if !ok {
		sess, err := b.api.CreateSession(ctx, &CreateSessionRequest{})
		if err != nil {
			b.reply(channelID, fmt.Sprintf("Failed to create session: %v", err))
			return
		}
		sessionID = sess.UUID
		b.sessions.Set(channelID, sessionID)
	}

	resp, err := b.api.SendMessage(ctx, sessionID, &MessageRequest{Role: "user", Content: decorateDiscordContext(user, content)})
	if err != nil {
		b.reply(channelID, fmt.Sprintf("Error: %v", err))
		return
	}

	if resp.Content != "" {
		b.reply(channelID, resp.Content)
	}
}

func (b *Bot) onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		b.handleApplicationCommand(i)
	}
}

func (b *Bot) registerCommands() error {
	commands := []*discordgo.ApplicationCommand{
		{
			Name: "ask", Description: "Ask the assistant a question",
			Options: []*discordgo.ApplicationCommandOption{{
				Type: discordgo.ApplicationCommandOptionString, Name: "prompt", Description: "Your question", Required: true,
			}},
		},
		{
			Name: "conversation", Description: "Start a conversation thread with a prompt",
			Options: []*discordgo.ApplicationCommandOption{{
				Type: discordgo.ApplicationCommandOptionString, Name: "prompt", Description: "Conversation starting prompt", Required: true,
			}},
		},
	}
	guildID := b.cfg.GuildID // empty = global
	for _, cmd := range commands {
		if _, err := b.dg.ApplicationCommandCreate(b.dg.State.User.ID, guildID, cmd); err != nil {
			return fmt.Errorf("cannot create '%s' command: %w", cmd.Name, err)
		}
	}
	return nil
}

func (b *Bot) unregisterCommands() error {
	guildID := b.cfg.GuildID
	cmds, err := b.dg.ApplicationCommands(b.dg.State.User.ID, guildID)
	if err != nil {
		return err
	}
	for _, c := range cmds {
		if c.Name == "ask" || c.Name == "conversation" {
			_ = b.dg.ApplicationCommandDelete(b.dg.State.User.ID, guildID, c.ID)
		}
	}
	return nil
}

func (b *Bot) handleApplicationCommand(i *discordgo.InteractionCreate) {
	name := i.ApplicationCommandData().Name
	switch name {
	case "ask":
		b.handleAsk(i)
	case "conversation":
		b.handleConversation(i)
	}
}

func (b *Bot) handleAsk(i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	prompt := ""
	if len(options) > 0 {
		prompt = options[0].StringValue()
	}
	if strings.TrimSpace(prompt) == "" {
		respondEphemeral(b.dg, i, "Please provide a prompt.")
		return
	}

	// Acknowledge and defer
	deferReply(b.dg, i, true)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		sess, err := b.api.CreateSession(ctx, &CreateSessionRequest{})
		if err != nil {
			editFollowup(b.dg, i, fmt.Sprintf("Failed to create session: %v", err))
			return
		}

		resp, err := b.api.SendMessage(ctx, sess.UUID, &MessageRequest{Role: "user", Content: prompt})
		if err != nil {
			editFollowup(b.dg, i, fmt.Sprintf("Error: %v", err))
			return
		}

		if resp.Content == "" {
			editFollowup(b.dg, i, "(no content)")
			return
		}
		editFollowup(b.dg, i, resp.Content)
	}()
}

func (b *Bot) handleConversation(i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	prompt := ""
	if len(options) > 0 {
		prompt = options[0].StringValue()
	}
	if strings.TrimSpace(prompt) == "" {
		respondEphemeral(b.dg, i, "Please provide a prompt.")
		return
	}

	if b.cfg.ConversationParentChannelID == "" {
		respondEphemeral(b.dg, i, "Conversation parent channel is not configured.")
		return
	}

	deferReply(b.dg, i, false)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		// Create new session for this thread
		sess, err := b.api.CreateSession(ctx, &CreateSessionRequest{})
		if err != nil {
			editFollowup(b.dg, i, fmt.Sprintf("Failed to create session: %v", err))
			return
		}

		// Seed conversation
		resp, err := b.api.SendMessage(ctx, sess.UUID, &MessageRequest{Role: "user", Content: prompt})
		if err != nil {
			editFollowup(b.dg, i, fmt.Sprintf("Error: %v", err))
			return
		}

		// Determine thread title: prefer resp.Summary, else truncate content
		title := strings.TrimSpace(resp.Summary)
		if title == "" {
			title = summarizeTitle(prompt, resp.Content)
		}
		if len(title) > 100 {
			title = title[:100]
		}

		// Create thread under the configured parent channel
		thread, err := b.dg.ThreadStartComplex(b.cfg.ConversationParentChannelID, &discordgo.ThreadStart{
			Name:                title,
			AutoArchiveDuration: 1440, // 24h
			Type:                discordgo.ChannelTypeGuildPublicThread,
		})
		if err != nil {
			editFollowup(b.dg, i, fmt.Sprintf("Failed to create thread: %v", err))
			return
		}

		// Bind session to thread
		b.sessions.Set(thread.ID, sess.UUID)

		// Post initial content in thread
		content := strings.TrimSpace(resp.Content)
		if content == "" {
			content = "(no content)"
		}
		_, _ = b.dg.ChannelMessageSend(thread.ID, content)

		editFollowup(b.dg, i, fmt.Sprintf("Created conversation thread: <#%s>", thread.ID))
	}()
}

func (b *Bot) reply(channelID, content string) {
	// Chunk content to Discord limits (2000 chars)
	for _, chunk := range chunkString(content, 1900) {
		_, _ = b.dg.ChannelMessageSend(channelID, chunk)
	}
}

// Utilities

func respondEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: content, Flags: discordgo.MessageFlagsEphemeral},
	})
}

func deferReply(s *discordgo.Session, i *discordgo.InteractionCreate, ephemeral bool) {
	flags := discordgo.MessageFlags(0)
	if ephemeral {
		flags = discordgo.MessageFlagsEphemeral
	}
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: flags},
	})
}

func editFollowup(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	// Chunking handled simply: send first chunk, then followups.
	chunks := chunkString(content, 1900)
	if len(chunks) == 0 {
		chunks = []string{""}
	}
	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &chunks[0]})
	for _, c := range chunks[1:] {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{Content: c})
	}
}

func chunkString(s string, size int) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []string
	for len(s) > size {
		// Try to split on paragraph or sentence boundaries
		split := findSplit(s[:size])
		out = append(out, strings.TrimSpace(s[:split]))
		s = s[split:]
	}
	if strings.TrimSpace(s) != "" {
		out = append(out, strings.TrimSpace(s))
	}
	return out
}

var splitRe = regexp.MustCompile(`(?s)(.*?[\n\r]{2}|.*?[.!?])$`)

func findSplit(s string) int {
	m := splitRe.FindStringSubmatchIndex(s)
	if len(m) >= 4 {
		return m[3]
	}
	return len(s)
}

func summarizeTitle(prompt, resp string) string {
	text := strings.TrimSpace(prompt)
	if text == "" {
		text = resp
	}
	text = strings.ReplaceAll(text, "\n", " ")
	if len(text) > 80 {
		text = text[:80] + "…"
	}
	return text
}

func decorateDiscordContext(user *discordgo.User, content string) string {
	uname := user.Username
	if user.GlobalName != "" {
		uname = user.GlobalName
	}
	return fmt.Sprintf("[discord user: %s] %s", uname, content)
}
