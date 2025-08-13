// agent.go handles declaring the memory agent struct
package memoryagent

import (
	"context"

	"github.com/ethanbaker/assistant/pkg/agent"
	"github.com/ethanbaker/assistant/pkg/memory"
	"github.com/ethanbaker/assistant/pkg/session"
	"github.com/ethanbaker/assistant/pkg/utils"
	"github.com/nlpodyssey/openai-agents-go/agents"
)

// MemoryAgent provides memory search and fact management capabilities
type MemoryAgent struct {
	agent        *agents.Agent
	config       *utils.Config
	memoryStore  *memory.Store
	sessionStore *session.Store
}

// NewMemoryAgent creates a new memory agent
func NewMemoryAgent(memoryStore *memory.Store, sessionStore *session.Store) *MemoryAgent {
	config := agent.LoadAgentConfig("memory-agent")

	// Create the underlying agent
	agentInstance := agents.New("memory-agent").
		WithInstructions("You are a memory assistant that helps store, retrieve, and search through information and past conversations.").
		WithModel("gpt-4o-mini")

	ma := &MemoryAgent{
		agent:        agentInstance,
		config:       config,
		memoryStore:  memoryStore,
		sessionStore: sessionStore,
	}

	// Register tools
	ma.registerTools()

	return ma
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
