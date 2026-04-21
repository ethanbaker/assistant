package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ethanbaker/assistant/pkg/sdk"
)

// onInteractionCreate dispatches slash command interactions.
func (b *Bot) onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionApplicationCommand {
		b.handleApplicationCommand(i)
	}
}

// registerCommands registers the bot's slash commands with Discord.
func (b *Bot) registerCommands() error {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "ask",
			Description: "Ask a one-off question",
			Options: []*discordgo.ApplicationCommandOption{{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "prompt",
				Description: "Your question",
				Required:    true,
			}},
		},
		{
			Name:        "conversation",
			Description: "Start a persistent conversation thread",
			Options: []*discordgo.ApplicationCommandOption{{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "prompt",
				Description: "Opening message for the conversation",
				Required:    true,
			}},
		},
	}

	for _, cmd := range commands {
		if _, err := b.dg.ApplicationCommandCreate(b.dg.State.User.ID, b.guildID, cmd); err != nil {
			return fmt.Errorf("cannot create %q command: %w", cmd.Name, err)
		}
	}
	return nil
}

// unregisterCommands removes the bot's slash commands from Discord.
func (b *Bot) unregisterCommands() error {
	cmds, err := b.dg.ApplicationCommands(b.dg.State.User.ID, b.guildID)
	if err != nil {
		return err
	}
	for _, c := range cmds {
		if c.Name == "ask" || c.Name == "conversation" {
			_ = b.dg.ApplicationCommandDelete(b.dg.State.User.ID, b.guildID, c.ID)
		}
	}
	return nil
}

func (b *Bot) handleApplicationCommand(i *discordgo.InteractionCreate) {
	if i == nil {
		return
	}
	switch i.ApplicationCommandData().Name {
	case "ask":
		b.handleAsk(i)
	case "conversation":
		b.handleConversation(i)
	}
}

// handleAsk processes the /ask command: creates an ephemeral one-off session.
func (b *Bot) handleAsk(i *discordgo.InteractionCreate) {
	prompt := firstStringOption(i)
	if prompt == "" {
		respondEphemeral(b.dg, i, "Please provide a prompt.")
		return
	}

	deferReply(b.dg, i, true) // ephemeral

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		sess, err := b.api.CreateSession(ctx, &sdk.CreateSessionRequest{
			UserID: i.Member.User.ID,
		})
		if err != nil {
			editFollowup(b.dg, i, fmt.Sprintf("Failed to create session: %v", err))
			return
		}

		resp, err := b.api.SendMessage(ctx, sess.ID, &sdk.PostMessageRequest{Content: prompt})
		if err != nil {
			editFollowup(b.dg, i, fmt.Sprintf("Failed to send message: %v", err))
			return
		}

		output := strings.TrimSpace(resp.FinalOutput)
		if output == "" {
			output = noContent
		}
		editFollowup(b.dg, i, output)
	}()
}

// handleConversation processes the /conversation command: creates a thread with a persistent session.
func (b *Bot) handleConversation(i *discordgo.InteractionCreate) {
	prompt := firstStringOption(i)
	if prompt == "" {
		respondEphemeral(b.dg, i, "Please provide a prompt.")
		return
	}

	if b.threadChannelID == "" {
		respondEphemeral(b.dg, i, "Thread channel is not configured.")
		return
	}

	if i.ChannelID != b.threadChannelID {
		respondEphemeral(b.dg, i, fmt.Sprintf("Please use <#%s> to start conversations.", b.threadChannelID))
		return
	}

	deferReply(b.dg, i, false)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		sess, err := b.api.CreateSession(ctx, &sdk.CreateSessionRequest{
			UserID: i.Member.User.ID,
		})
		if err != nil {
			editFollowup(b.dg, i, fmt.Sprintf("Failed to create session: %v", err))
			return
		}

		resp, err := b.api.SendMessage(ctx, sess.ID, &sdk.PostMessageRequest{Content: prompt})
		if err != nil {
			editFollowup(b.dg, i, fmt.Sprintf("Failed to send message: %v", err))
			return
		}

		title := fmt.Sprintf("Conversation %s", sess.ID[:8])
		thread, err := b.dg.ThreadStartComplex(b.threadChannelID, &discordgo.ThreadStart{
			Name:                title,
			AutoArchiveDuration: threadArchiveDuration,
			Type:                discordgo.ChannelTypeGuildPublicThread,
		})
		if err != nil {
			editFollowup(b.dg, i, fmt.Sprintf("Failed to create thread: %v", err))
			return
		}

		b.conversations.Set(thread.ID, sess.ID)

		output := strings.TrimSpace(resp.FinalOutput)
		if output == "" {
			output = noContent
		}
		reply(b.dg, thread.ID, output)
		editFollowup(b.dg, i, fmt.Sprintf("Started conversation in <#%s>.", thread.ID))
	}()
}

// firstStringOption extracts the first string option value from an interaction.
func firstStringOption(i *discordgo.InteractionCreate) string {
	opts := i.ApplicationCommandData().Options
	if len(opts) == 0 {
		return ""
	}
	return strings.TrimSpace(opts[0].StringValue())
}
