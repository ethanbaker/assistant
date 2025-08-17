package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// respondEphemeral sends a response that is only visible to the user who invoked the command
func respondEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: content, Flags: discordgo.MessageFlagsEphemeral},
	})
}

// deferReply sends a deferred response to acknowledge the interaction
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

// editFollowup edits the initial response to an interaction with new content
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

// chunkString splits a long string into smaller chunks, ensuring no chunk exceeds the specified size
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

// splitRe is a regex to find natural split points in text
var splitRe = regexp.MustCompile(`(?s)(.*?[\n\r]{2}|.*?[.!?])$`)

// findSplit finds the index of a good split point in the string
func findSplit(s string) int {
	m := splitRe.FindStringSubmatchIndex(s)
	if len(m) >= 4 {
		return m[3]
	}
	return len(s)
}

// decorateDiscordContext formats the user context for Discord messages to include the user's name for model context
func decorateDiscordContext(user *discordgo.User, content string) string {
	uname := user.Username
	if user.GlobalName != "" {
		uname = user.GlobalName
	}
	return fmt.Sprintf("[discord user: %s] %s", uname, content)
}
