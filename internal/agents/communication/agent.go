package communication

import (
	"context"
	"errors"
	"time"

	"github.com/ethanbaker/assistant/internal/stores/memory"
	"github.com/ethanbaker/assistant/internal/stores/session"
	"github.com/ethanbaker/assistant/pkg/agent"
	"github.com/ethanbaker/assistant/pkg/utils"
	"github.com/nlpodyssey/openai-agents-go/agents"
)

// CommunicationAgent provides communication and messaging capabilities
type CommunicationAgent struct {
	agent        *agents.Agent
	config       *utils.Config
	memoryStore  *memory.Store
	sessionStore session.Store
	basePrompt   string
}

// NewCommunicationAgent creates a new communication agent
func NewCommunicationAgent(memoryStore *memory.Store, sessionStore session.Store, config *utils.Config) (*CommunicationAgent, error) {
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
	var err error
	ca.basePrompt, err = utils.LoadPrompt(path)
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
	ca.agent = agents.New("communication-agent").
		WithModel(config.Get("MODEL")).
		WithMCPServers(mcpServers).
		WithInstructionsFunc(ca.getPrompt)

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

// getPrompt returns the prompt for the agent
func (ca *CommunicationAgent) getPrompt(ctx context.Context, a *agents.Agent) (string, error) {
	now := time.Now()

	builder := agent.NewPromptBuilder(ca.basePrompt)
	builder.AddContext("Current time: " + now.Format("15:04:05 MST"))
	builder.AddContext("Today's date: " + now.Format("Monday, 2006-01-02"))

	return builder.Build(), nil
}
