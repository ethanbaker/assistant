// tools.go handles registering tools for the CommunicationAgent
package communication

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/nlpodyssey/openai-agents-go/agents"
	"github.com/openai/openai-go/v2/packages/param"
)

// registerTools registers the communication-related tools
func (ca *CommunicationAgent) registerTools() {
	// Summarize content tool
	summarizeTool := agents.FunctionTool{
		Name:        "summarize_content",
		Description: "Summarize text content for communication purposes",
		ParamsJSONSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"content": map[string]any{
					"type":        "string",
					"description": "The content to summarize",
				},
				"max_length": map[string]any{
					"type":        "integer",
					"description": "Maximum length of the summary in words (optional, defaults to 100)",
				},
				"style": map[string]any{
					"type":        "string",
					"description": "Summary style: 'brief', 'detailed', or 'bullet_points' (optional, defaults to 'brief')",
					"enum":        []string{"brief", "detailed", "bullet_points"},
				},
			},
			"additionalProperties": false,
			"required":             []string{"content"},
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return ca.handleSummarizeContent(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}

	// Send Telegram message tool
	sendTelegramTool := agents.FunctionTool{
		Name:        "send_telegram_message",
		Description: "Send a message to a Telegram chat",
		ParamsJSONSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"chat_id": map[string]any{
					"type":        "string",
					"description": "The Telegram chat ID to send the message to",
				},
				"message": map[string]any{
					"type":        "string",
					"description": "The message text to send",
				},
				"parse_mode": map[string]any{
					"type":        "string",
					"description": "Parse mode for the message: 'Markdown' or 'HTML' (optional)",
					"enum":        []string{"Markdown", "HTML"},
				},
			},
			"additionalProperties": false,
			"required":             []string{"chat_id", "message"},
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return ca.handleSendTelegramMessage(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}

	// Format message tool
	formatMessageTool := agents.FunctionTool{
		Name:        "format_message",
		Description: "Format a message for a specific platform",
		ParamsJSONSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"content": map[string]any{
					"type":        "string",
					"description": "The content to format",
				},
				"platform": map[string]any{
					"type":        "string",
					"description": "Target platform: 'telegram', 'slack', 'email', or 'generic'",
					"enum":        []string{"telegram", "slack", "email", "generic"},
				},
				"style": map[string]any{
					"type":        "string",
					"description": "Formatting style: 'formal', 'informal', or 'technical' (optional, defaults to 'informal')",
					"enum":        []string{"formal", "informal", "technical"},
				},
			},
			"additionalProperties": false,
			"required":             []string{"content", "platform"},
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return ca.handleFormatMessage(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}

	// Add tools to agent
	ca.agent.Tools = []agents.Tool{
		summarizeTool,
		sendTelegramTool,
		formatMessageTool,
	}
}

// handleSummarizeContent handles content summarization
func (ca *CommunicationAgent) handleSummarizeContent(ctx context.Context, arguments string) (string, error) {
	if ca.ShouldDryRun(ctx) {
		return fmt.Sprintf("DRY RUN: Would summarize content with arguments: %s", arguments), nil
	}

	// Unmarshal the arguments
	var args struct {
		Content   string `json:"content"`
		MaxLength int    `json:"max_length,omitempty"`
		Style     string `json:"style,omitempty"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Set defaults
	if args.MaxLength == 0 {
		args.MaxLength = 100
	}
	if args.Style == "" {
		args.Style = "brief"
	}

	// This is a simplified implementation - in practice, you'd use an LLM or NLP library
	words := strings.Fields(args.Content)
	if len(words) <= args.MaxLength {
		return args.Content, nil
	}

	summary := strings.Join(words[:args.MaxLength], " ")
	if args.Style == "bullet_points" {
		lines := strings.Split(summary, ". ")
		summary = "• " + strings.Join(lines, "\n• ")
	}

	return summary + "...", nil
}

// handleSendTelegramMessage handles sending messages to Telegram
func (ca *CommunicationAgent) handleSendTelegramMessage(ctx context.Context, arguments string) (string, error) {
	if ca.ShouldDryRun(ctx) {
		return fmt.Sprintf("DRY RUN: Would send Telegram message with arguments: %s", arguments), nil
	}

	// Unmarshal the arguments
	var args struct {
		ChatID    string `json:"chat_id"`
		Message   string `json:"message"`
		ParseMode string `json:"parse_mode,omitempty"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Get bot token from config
	botToken := ca.config.Get("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		return "", fmt.Errorf("TELEGRAM_BOT_TOKEN not configured")
	}

	// Construct the Telegram API URL
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	// Prepare the request body
	requestBody := map[string]string{
		"chat_id": args.ChatID,
		"text":    args.Message,
	}
	if args.ParseMode != "" {
		requestBody["parse_mode"] = args.ParseMode
	}

	// Convert to JSON
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Make the HTTP request
	resp, err := http.Post(url, "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		return "", fmt.Errorf("failed to send Telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Telegram API returned status code: %d", resp.StatusCode)
	}

	return fmt.Sprintf("Message sent successfully to chat %s", args.ChatID), nil
}

// handleFormatMessage handles message formatting for different platforms
func (ca *CommunicationAgent) handleFormatMessage(ctx context.Context, arguments string) (string, error) {
	if ca.ShouldDryRun(ctx) {
		return fmt.Sprintf("DRY RUN: Would format message with arguments: %s", arguments), nil
	}

	// Unmarshal the arguments
	var args struct {
		Content  string `json:"content"`
		Platform string `json:"platform"`
		Style    string `json:"style,omitempty"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Set default style
	if args.Style == "" {
		args.Style = "informal"
	}

	content := args.Content

	// Apply platform-specific formatting
	switch args.Platform {
	case "telegram":
		if args.Style == "formal" {
			content = "📋 " + content
		} else {
			content = "💬 " + content
		}
	case "slack":
		if args.Style == "technical" {
			content = "```\n" + content + "\n```"
		}
	case "email":
		if args.Style == "formal" {
			content = "Dear recipient,\n\n" + content + "\n\nBest regards"
		}
	case "generic":
		// No special formatting for generic platform
	default:
		return "", fmt.Errorf("unsupported platform: %s", args.Platform)
	}

	return content, nil
}
