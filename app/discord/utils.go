package main

import (
	"fmt"
	"log"
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

// reply sends a message to the specified channel, chunking if necessary
func reply(s *discordgo.Session, channelID, content string) {
	// Chunk content to Discord limits (2000 chars)
	for _, chunk := range chunkString(content, 1900) {
		_, _ = s.ChannelMessageSend(channelID, chunk)
	}
}

// replySanitizeHTML sends a message to the specified channel, sanitizing HTML to Discord markdown and chunking if necessary
func replySanitizeHTML(s *discordgo.Session, channelID, content string) {
	sanitized := sanitizeHTMLToDiscordMarkdown(content)
	reply(s, channelID, sanitized)
}

// errorReply sends a formated error message to the specified channel
func errorReply(s *discordgo.Session, channelID, desc string, errs ...any) {
	// Create error message
	errorMsg := ""
	for _, e := range errs {
		errorMsg += fmt.Sprintf("\n > %v\n\n", e)
	}

	// Send to channel
	output := fmt.Sprintf("Error: %s %s", desc, errorMsg)
	reply(s, channelID, output)
	log.Printf("[DISCORD]: %s", output)
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

// Define HTML tag to Discord markdown mappings
var replacements = map[string][2]string{
	// Bold tags
	"strong": {"**", "**"},
	"b":      {"**", "**"},

	// Italic tags
	"em": {"*", "*"},
	"i":  {"*", "*"},

	// Underline tags (Discord uses __ for underline)
	"u": {"__", "__"},

	// Strikethrough tags
	"s":      {"~~", "~~"},
	"strike": {"~~", "~~"},
	"del":    {"~~", "~~"},

	// Blockquote tags
	"blockquote": {"\n> ", "\n"},
}

// sanitizeHTMLToDiscordMarkdown converts HTML elements to Discord markdown
func sanitizeHTMLToDiscordMarkdown(content string) string {
	result := content

	// Process each tag type
	for tag, markdown := range replacements {
		// Create case-insensitive regex patterns for opening and closing tags
		// Use word boundaries to prevent partial matches (e.g., <s> matching <strong>)
		openPattern := fmt.Sprintf(`(?i)<%s(\s[^>]*|/?)>`, tag)
		closePattern := fmt.Sprintf(`(?i)</%s>`, tag)

		// Replace opening tags with opening markdown
		openRe := regexp.MustCompile(openPattern)
		result = openRe.ReplaceAllString(result, markdown[0])

		// Replace closing tags with closing markdown
		closeRe := regexp.MustCompile(closePattern)
		result = closeRe.ReplaceAllString(result, markdown[1])
	}

	// Handle special cases for code blocks with language specification
	preCodeRe := regexp.MustCompile(`(?i)<pre><code[^>]*class="language-([^"]*)"[^>]*>`)
	result = preCodeRe.ReplaceAllStringFunc(result, func(match string) string {
		langMatch := regexp.MustCompile(`(?i)class="language-([^"]*)"`)
		langMatches := langMatch.FindStringSubmatch(match)
		if len(langMatches) > 1 {
			return fmt.Sprintf("```%s\n", langMatches[1])
		}
		return "```"
	})

	// Handle preformatted text without language specification
	result = regexp.MustCompile(`(?i)<pre><code>`).ReplaceAllString(result, "```\n")
	result = regexp.MustCompile(`(?i)</code></pre>`).ReplaceAllString(result, "\n```")

	// Now check inline code tags
	inlineCodeRe := regexp.MustCompile(`(?i)<code>(.*?)</code>`)
	result = inlineCodeRe.ReplaceAllString(result, "`$1`")

	// Clean up any remaining HTML tags that weren't converted
	htmlTagRe := regexp.MustCompile(`<[^>]*>`)
	result = htmlTagRe.ReplaceAllString(result, "")

	// Clean up extra whitespace
	result = strings.TrimSpace(result)

	// Handle escape sequences for Discord markdown
	result = strings.ReplaceAll(result, "\\n", `
	`)

	return result
}
