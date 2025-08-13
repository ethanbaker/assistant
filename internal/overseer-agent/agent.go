package overseeragent

import (
	"context"
	"log"

	memoryagent "github.com/ethanbaker/assistant/internal/memory-agent"
	"github.com/ethanbaker/assistant/pkg/agent"
	"github.com/ethanbaker/assistant/pkg/memory"
	"github.com/ethanbaker/assistant/pkg/session"
	"github.com/ethanbaker/assistant/pkg/utils"
	"github.com/nlpodyssey/openai-agents-go/agents"
)

// OverseerAgent coordinates and hands off to specialized agents
type OverseerAgent struct {
	agent        *agents.Agent
	config       *utils.Config
	memoryStore  *memory.Store
	sessionStore *session.Store
}

// NewOverseerAgent creates a new overseer agent with handoffs to all specialized agents
func NewOverseerAgent(memoryStore *memory.Store, sessionStore *session.Store) *OverseerAgent {
	config := agent.LoadAgentConfig("overseer-agent")

	// Create specialized agents for handoffs
	memoryAgent := memoryagent.NewMemoryAgent(memoryStore, sessionStore)

	// Create handoffs for each specialized agent
	memoryHandoff := agents.HandoffFromAgent(agents.HandoffFromAgentParams{
		Agent:                   memoryAgent.Agent(),
		ToolNameOverride:        "handoff_to_memory_agent",
		ToolDescriptionOverride: "Hand off to the Memory Agent for storing facts, recalling information, or searching past conversations",
	})

	// Load instructions from file with fallback to hardcoded version
	instructions, err := utils.LoadPrompt("prompts/overseer-agent.txt")
	if err != nil {
		log.Fatalf("Failed to load overseer agent instructions: %v", err)
	}

	// Create the overseer agent with handoffs
	agentInstance := agents.New("overseer-agent").
		WithInstructions(instructions).
		WithModel("gpt-4o-mini").
		WithHandoffs(
			memoryHandoff,
		)

	oa := &OverseerAgent{
		agent:        agentInstance,
		config:       config,
		memoryStore:  memoryStore,
		sessionStore: sessionStore,
	}

	return oa
}

// Agent returns the underlying openai-agents-go instance
func (oa *OverseerAgent) Agent() *agents.Agent {
	return oa.agent
}

// ID returns the agent identifier
func (oa *OverseerAgent) ID() string {
	return "overseer-agent"
}

// Config returns the agent configuration
func (oa *OverseerAgent) Config() *utils.Config {
	return oa.config
}

// ShouldDryRun determines if the agent should run in dry-run mode
func (oa *OverseerAgent) ShouldDryRun(ctx context.Context) bool {
	return oa.config.GetBool("DRY_RUN")
}
