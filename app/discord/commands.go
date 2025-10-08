package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ethanbaker/assistant/pkg/sdk"
)

// onInteractionCreate handles interactions (slash commands)
func (b *Bot) onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		b.handleApplicationCommand(i)
	}
}

// registerCommands registers the bot's slash commands with Discord
func (b *Bot) registerCommands() error {
	// Define commands
	commands := []*discordgo.ApplicationCommand{
		{
			Name: "ask", Description: "Ask me a question",
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

	// Register commands
	guildID := b.guildID // empty = global commands
	for _, cmd := range commands {
		if _, err := b.dg.ApplicationCommandCreate(b.dg.State.User.ID, guildID, cmd); err != nil {
			return fmt.Errorf("cannot create '%s' command: %w", cmd.Name, err)
		}
	}

	return nil
}

// unregisterCommands removes the bot's slash commands from Discord
func (b *Bot) unregisterCommands() error {
	guildID := b.guildID
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

// handleApplicationCommand processes a slash command interaction
func (b *Bot) handleApplicationCommand(i *discordgo.InteractionCreate) {
	if i == nil {
		return
	}

	name := i.ApplicationCommandData().Name

	switch name {
	case "ask":
		b.handleAsk(i)
	case "conversation":
		b.handleConversation(i)
	}
}

// handleAsk handles the "ask" command interaction
func (b *Bot) handleAsk(i *discordgo.InteractionCreate) {
	// Extract the prompt from first provided option
	options := i.ApplicationCommandData().Options
	prompt := ""
	if len(options) > 0 {
		prompt = options[0].StringValue()
	}

	if strings.TrimSpace(prompt) == "" {
		respondEphemeral(b.dg, i, "Please provide a prompt")
		return
	}

	// Acknowledge creation and defer
	deferReply(b.dg, i, true)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		// Create new session for this interaction
		sess, err := b.api.CreateSession(ctx, &sdk.CreateSessionRequest{
			UserID: i.Member.User.ID,
		})
		if err != nil {
			editFollowup(b.dg, i, fmt.Sprintf("Failed to create session: %v", err))
			return
		}

		// Send the message to the session
		resp, err := b.api.SendMessage(ctx, sess.ID, &sdk.PostMessageRequest{Content: prompt})
		if err != nil {
			editFollowup(b.dg, i, fmt.Sprintf("Failed to send message: %v", err))
			return
		}

		// Send final output
		output := strings.TrimSpace(resp.FinalOutput)
		if output == "" {
			editFollowup(b.dg, i, NO_CONTENT)
		} else {
			editFollowup(b.dg, i, output)
		}
	}()
}

// handleConversation handles the "conversation" command interaction
func (b *Bot) handleConversation(i *discordgo.InteractionCreate) {
	// Extract the prompt from first provided option
	options := i.ApplicationCommandData().Options
	prompt := ""
	if len(options) > 0 {
		prompt = options[0].StringValue()
	}

	if strings.TrimSpace(prompt) == "" {
		respondEphemeral(b.dg, i, "Please provide a prompt.")
		return
	}

	// Make sure thread channel is configured
	if b.threadChannelID == "" {
		respondEphemeral(b.dg, i, "Thread channel is not configured.")
		return
	}

	// Make sure this channel is the thread channel
	if i.ChannelID != b.threadChannelID {
		respondEphemeral(b.dg, i, fmt.Sprintf("Please use the <#%s> channel for conversations.", b.threadChannelID))
		return
	}

	// Acknowledge creation and defer
	deferReply(b.dg, i, false)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		// Create new session for this thread
		sess, err := b.api.CreateSession(ctx, &sdk.CreateSessionRequest{
			UserID: i.Member.User.ID,
		})
		if err != nil {
			editFollowup(b.dg, i, fmt.Sprintf("Failed to create session: %v", err))
			return
		}

		// Seed conversation
		resp, err := b.api.SendMessage(ctx, sess.ID, &sdk.PostMessageRequest{
			Content: prompt,
		})
		if err != nil {
			editFollowup(b.dg, i, fmt.Sprintf("Failed to send message: %v", err))
			return
		}

		// Determine thread title: prefer resp.Summary, else truncate content
		// TODO: call backend endpoint to summarize
		title := fmt.Sprintf("Conversation %s", sess.ID[:4])

		// Create thread under the configured parent channel
		thread, err := b.dg.ThreadStartComplex(b.threadChannelID, &discordgo.ThreadStart{
			Name:                title,
			AutoArchiveDuration: THREAD_ARCHIVE,
			Type:                discordgo.ChannelTypeGuildPublicThread,
		})
		if err != nil {
			editFollowup(b.dg, i, fmt.Sprintf("Failed to create thread: %v", err))
			return
		}

		// Bind conversation to thread
		b.conversations.Set(thread.ID, sess.ID)

		// Post initial content in thread
		output := strings.TrimSpace(resp.FinalOutput)
		if output == "" {
			editFollowup(b.dg, i, NO_CONTENT)
		} else {
			reply(b.dg, thread.ID, output)
		}

		editFollowup(b.dg, i, fmt.Sprintf("Created conversation thread: <%4s>", sess.ID[:4]))
	}()
}
