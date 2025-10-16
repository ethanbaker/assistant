// agent.go handles declaring the memory agent struct
package memory

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

// MemoryAgent provides memory search and fact management capabilities
type MemoryAgent struct {
	agent        *agents.Agent
	config       *utils.Config
	memoryStore  *memory.Store
	sessionStore session.Store
	basePrompt   string
}

// NewMemoryAgent creates a new memory agent
func NewMemoryAgent(memoryStore *memory.Store, sessionStore session.Store, config *utils.Config) (*MemoryAgent, error) {
	// Get sysprompt path
	path := config.Get("MEMORY_SYSPROMPT_PATH")
	if path == "" {
		return nil, errors.New("MEMORY_SYSPROMPT_PATH not set in environment")
	}

	ma := &MemoryAgent{
		config:       config,
		memoryStore:  memoryStore,
		sessionStore: sessionStore,
	}

	// Load instructions from file with fallback to hardcoded version
	var err error
	ma.basePrompt, err = utils.LoadPrompt(path)
	if err != nil {
		return nil, err
	}

	// Create the underlying agent
	ma.agent = agents.New("memory-agent").
		WithModel(config.Get("MODEL")).
		WithInstructionsFunc(ma.getPrompt)

	// Register tools
	ma.registerTools()

	return ma, nil
}

// Agent returns the underlying openai-agents-go instance
func (ma *MemoryAgent) Agent() *agents.Agent {
	return ma.agent
}

// ID returns the agent identifier
func (ma *MemoryAgent) ID() string {
	return "memory-agent"
}

// Config returns the agent configuration
func (ma *MemoryAgent) Config() *utils.Config {
	return ma.config
}

// ShouldDryRun determines if the agent should run in dry-run mode
func (ma *MemoryAgent) ShouldDryRun(ctx context.Context) bool {
	return ma.config.GetBool("DRY_RUN")
}

// getPrompt returns the prompt for the agent
func (ma *MemoryAgent) getPrompt(ctx context.Context, a *agents.Agent) (string, error) {
	now := time.Now()

	builder := agent.NewPromptBuilder(ma.basePrompt)
	builder.AddContext("Current time: " + now.Format("15:04:05 MST"))
	builder.AddContext("Today's date: " + now.Format("Monday, 2006-01-02"))

	return builder.Build(), nil
}
