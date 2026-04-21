package communication

import (
	"context"
	"errors"
	"os"

	"github.com/ethanbaker/assistant/internal/prompts"
	"github.com/ethanbaker/assistant/pkg/config"
	"github.com/nlpodyssey/openai-agents-go/agents"
)

// CommunicationAgentConfig provided on construct
type CommunicationAgentConfig struct {
	Model               string
	PromptFile          string
	TelegramAppID       string
	TelegramAPIHash     string
	TelegramSessionFile string
}

// CommunicationAgent provides communication and messaging capabilities
type CommunicationAgent struct {
	agent      *agents.Agent
	basePrompt string

	telegramAppID       string
	telegramAPIHash     string
	telegramSessionFile string
}

// NewCommunicationAgent creates a new communication agent
func NewCommunicationAgent(cfg CommunicationAgentConfig) (*CommunicationAgent, error) {
	if cfg.Model == "" {
		return nil, errors.New("Model is required")
	}
	if cfg.PromptFile == "" {
		return nil, errors.New("Prompt file path is required")
	}
	if cfg.TelegramAppID == "" {
		return nil, errors.New("TelegramAppID is required")
	}
	if cfg.TelegramAPIHash == "" {
		return nil, errors.New("TelegramAPIHash is required")
	}
	if cfg.TelegramSessionFile == "" {
		return nil, errors.New("TelegramSessionFile is required")
	}

	ca := &CommunicationAgent{
		telegramAppID:       cfg.TelegramAppID,
		telegramAPIHash:     cfg.TelegramAPIHash,
		telegramSessionFile: cfg.TelegramSessionFile,
	}

	data, err := os.ReadFile(cfg.PromptFile)
	if err != nil {
		return nil, err
	}
	ca.basePrompt = string(data)

	// Create MCP servers
	telegramMCP, err := ca.getTelegramMCP()
	if err != nil {
		return nil, err
	}

	mcpServers := []agents.MCPServer{
		telegramMCP,
	}

	// Create the underlying agent
	ca.agent = agents.New("communication-agent").
		WithModel(cfg.Model).
		WithMCPServers(mcpServers).
		WithInstructionsFunc(ca.getPrompt)

	ca.registerTools()

	return ca, nil
}

// Agent returns the underlying openai-agents-go instance
func (ca *CommunicationAgent) Agent() *agents.Agent {
	return ca.agent
}

// ID returns the agent identifier
func (ca *CommunicationAgent) ID() string {
	return "communication-agent"
}

// ShouldDryRun determines if the agent should run in dry-run mode
func (ca *CommunicationAgent) ShouldDryRun(ctx context.Context) bool {
	return config.GetenvValue("DRY_RUN") == "true"
}

// getPrompt returns the prompt for the agent
func (ca *CommunicationAgent) getPrompt(ctx context.Context, a *agents.Agent) (string, error) {
	builder := prompts.NewPromptBuilder(ca.basePrompt)
	return builder.Build(), nil
}
