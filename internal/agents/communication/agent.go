package communication

import (
	"context"
	"errors"

	"github.com/ethanbaker/assistant/internal/stores/memory"
	"github.com/ethanbaker/assistant/internal/stores/session"
	"github.com/ethanbaker/assistant/pkg/utils"
	"github.com/nlpodyssey/openai-agents-go/agents"
)

// CommunicationAgent provides communication and messaging capabilities
type CommunicationAgent struct {
	agent        *agents.Agent
	config       *utils.Config
	memoryStore  *memory.Store
	sessionStore *session.Store
}

// NewCommunicationAgent creates a new communication agent
func NewCommunicationAgent(memoryStore *memory.Store, sessionStore *session.Store, config *utils.Config) (*CommunicationAgent, error) {
	ca := &CommunicationAgent{
		config:       config,
		memoryStore:  memoryStore,
		sessionStore: sessionStore,
	}

	// Get sysprompt path
	path := config.Get("COMMUNICATION_SYSPROMPT_PATH")
	if path == "" {
		return nil, errors.New("COMMUNICATION_SYSPROMPT_PATH not set in environment")
	}

	// Load instructions from file with fallback to hardcoded version
	instructions, err := utils.LoadPrompt(path)
	if err != nil {
		return nil, err
	}

	// Create MCP servers
	telegramMCP, err := ca.getTelegramMCP()
	if err != nil {
		return nil, err
	}

	mcpServers := []agents.MCPServer{
		telegramMCP,
	}

	// Create the underlying agent
	agentInstance := agents.New("communication-agent").
		WithInstructions(instructions).
		WithModel(config.Get("MODEL")).
		WithMCPServers(mcpServers)

	ca.agent = agentInstance

	// Register tools
	//ca.registerTools()

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

// Config returns the agent configuration
func (ca *CommunicationAgent) Config() *utils.Config {
	return ca.config
}

// ShouldDryRun determines if the agent should run in dry-run mode
func (ca *CommunicationAgent) ShouldDryRun(ctx context.Context) bool {
	return ca.config.GetBool("DRY_RUN")
}
