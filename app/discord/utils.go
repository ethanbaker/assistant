package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/ethanbaker/assistant/pkg/logger"
)

// respondEphemeral sends a response visible only to the user who invoked the command.
func respondEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// deferReply acknowledges the interaction with a deferred response.
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

// editFollowup edits the deferred interaction response. Chunks content if it exceeds Discord limits.
func editFollowup(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	chunks := chunkString(content, 1900)
	if len(chunks) == 0 {
		chunks = []string{""}
	}
	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &chunks[0]})
	for _, c := range chunks[1:] {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{Content: c})
	}
}

// reply sends a message to a channel, chunking if the content exceeds Discord's 2000-character limit.
func reply(s *discordgo.Session, channelID, content string) {
	for _, chunk := range chunkString(content, 1900) {
		_, _ = s.ChannelMessageSend(channelID, chunk)
	}
}

// replySanitizeHTML sanitizes HTML to Discord markdown before sending.
func replySanitizeHTML(s *discordgo.Session, channelID, content string) {
	reply(s, channelID, sanitizeHTMLToDiscordMarkdown(content))
}

// errorReply sends a formatted error message to a channel and logs it.
func errorReply(s *discordgo.Session, channelID, desc string, errs ...any) {
	var errorMsg strings.Builder
	for _, e := range errs {
		fmt.Fprintf(&errorMsg, "\n > %v\n\n", e)
	}

	output := fmt.Sprintf("Error: %s %s", desc, errorMsg.String())
	reply(s, channelID, output)
	logger.Errorf("%s", output)
}

// chunkString splits a string into chunks no larger than size, splitting on natural boundaries.
func chunkString(s string, size int) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	var out []string
	for len(s) > size {
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

// findSplit finds a good split index (paragraph or sentence boundary).
func findSplit(s string) int {
	m := splitRe.FindStringSubmatchIndex(s)
	if len(m) >= 4 {
		return m[3]
	}
	return len(s)
}

// decorateDiscordContext prepends Discord user context to a message for the agent.
func decorateDiscordContext(user *discordgo.User, content string) string {
	name := user.Username
	if user.GlobalName != "" {
		name = user.GlobalName
	}
	return fmt.Sprintf("[discord user: %s] %s", name, content)
}

// replacements maps HTML tags to their Discord markdown equivalents.
var replacements = map[string][2]string{
	"strong":     {"**", "**"},
	"b":          {"**", "**"},
	"em":         {"*", "*"},
	"i":          {"*", "*"},
	"u":          {"__", "__"},
	"s":          {"~~", "~~"},
	"strike":     {"~~", "~~"},
	"del":        {"~~", "~~"},
	"blockquote": {"\n> ", "\n"},
}

// sanitizeHTMLToDiscordMarkdown converts HTML elements to Discord markdown.
func sanitizeHTMLToDiscordMarkdown(content string) string {
	result := content

	for tag, markdown := range replacements {
		openPattern := fmt.Sprintf(`(?i)<%s(\s[^>]*|/?)>`, tag)
		closePattern := fmt.Sprintf(`(?i)</%s>`, tag)

		openRe := regexp.MustCompile(openPattern)
		result = openRe.ReplaceAllString(result, markdown[0])

		closeRe := regexp.MustCompile(closePattern)
		result = closeRe.ReplaceAllString(result, markdown[1])
	}

	// Code blocks with language hint
	preCodeRe := regexp.MustCompile(`(?i)<pre><code[^>]*class="language-([^"]*)"[^>]*>`)
	result = preCodeRe.ReplaceAllStringFunc(result, func(match string) string {
		langMatch := regexp.MustCompile(`(?i)class="language-([^"]*)"`)
		langMatches := langMatch.FindStringSubmatch(match)
		if len(langMatches) > 1 {
			return fmt.Sprintf("\n```%s\n", langMatches[1])
		}
		return "\n```\n"
	})

	// Plain code blocks
	result = regexp.MustCompile(`(?i)<pre><code>`).ReplaceAllString(result, "\n```\n")
	result = regexp.MustCompile(`(?i)</code></pre>`).ReplaceAllString(result, "\n```\n")

	// Inline code
	inlineCodeRe := regexp.MustCompile(`(?i)<code>(.*?)</code>`)
	result = inlineCodeRe.ReplaceAllString(result, "`$1`")

	// Strip remaining HTML tags
	result = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(result, "")

	result = strings.TrimSpace(result)
	result = strings.ReplaceAll(result, "\\n", "\n")

	return result
}
