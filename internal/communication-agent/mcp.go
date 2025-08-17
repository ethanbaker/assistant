package communicationagent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/nlpodyssey/openai-agents-go/agents"
)

/** Helper functions */
func (ca *CommunicationAgent) getTelegramMCP() (agents.MCPServer, error) {
	// Get config variables
	appId := ca.config.Get("TG_APP_ID")
	if appId == "" {
		return nil, errors.New("TG_APP_ID is not set in environment")
	}

	apiHash := ca.config.Get("TG_API_HASH")
	if apiHash == "" {
		return nil, errors.New("TG_API_HASH is not set in environment")
	}

	sessionFile := ca.config.Get("TG_SESSION_PATH")
	if sessionFile == "" {
		return nil, errors.New("TG_SESSION_PATH is not set in environment")
	}

	// Check if the session file exists
	if _, err := os.Stat(sessionFile); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("session file %s does not exist", sessionFile)
	}

	// Create MCP server for Telegram with session support
	server := agents.NewMCPServerStdio(agents.MCPServerStdioParams{
		CacheToolsList: true,
		Command:        exec.Command("npx", "-y", "@chaindead/telegram-mcp", "--app-id", appId, "--api-hash", apiHash, "--session", sessionFile),
	})

	// Run MCP server
	err := server.Connect(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to start Telegram MCP server: %w", err)
	}
	return server, nil

}
