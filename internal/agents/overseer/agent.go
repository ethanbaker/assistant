package overseer

import (
	"context"
	"errors"

	communicationagent "github.com/ethanbaker/assistant/internal/agents/communication"
	memoryagent "github.com/ethanbaker/assistant/internal/agents/memory"
	searchagent "github.com/ethanbaker/assistant/internal/agents/search"
	taskagent "github.com/ethanbaker/assistant/internal/agents/task"
	"github.com/ethanbaker/assistant/internal/stores/memory"
	"github.com/ethanbaker/assistant/internal/stores/session"
	"github.com/ethanbaker/assistant/pkg/utils"
	"github.com/nlpodyssey/openai-agents-go/agents"
)

// OverseerAgent coordinates and hands off to specialized agents
type OverseerAgent struct {
	agent        *agents.Agent
	config       *utils.Config
	memoryStore  *memory.Store
	sessionStore session.Store
}

// NewOverseerAgent creates a new overseer agent with handoffs to all specialized agents
func NewOverseerAgent(memoryStore *memory.Store, sessionStore session.Store, config *utils.Config) (*OverseerAgent, error) {
	// Create specialized agents for handoffs
	memoryAgent, err := memoryagent.NewMemoryAgent(memoryStore, sessionStore, config)
	if err != nil {
		return nil, err
	}

	communicationAgent, err := communicationagent.NewCommunicationAgent(memoryStore, sessionStore, config)
	if err != nil {
		return nil, err
	}

	searchAgent, err := searchagent.NewSearchAgent(memoryStore, sessionStore, config)
	if err != nil {
		return nil, err
	}

	taskAgent, err := taskagent.NewTaskAgent(memoryStore, sessionStore, config)
	if err != nil {
		return nil, err
	}

	// Create handoffs for each specialized agent
	memoryHandoff := agents.HandoffFromAgent(agents.HandoffFromAgentParams{
		Agent:                   memoryAgent.Agent(),
		ToolNameOverride:        "handoff_to_memory_agent",
		ToolDescriptionOverride: "Hand off to the Memory Agent for storing facts, recalling information, or searching past conversations",
	})

	communicationHandoff := agents.HandoffFromAgent(agents.HandoffFromAgentParams{
		Agent:                   communicationAgent.Agent(),
		ToolNameOverride:        "handoff_to_communication_agent",
		ToolDescriptionOverride: "Hand off to the Communication Agent for sending messages, summarizing content from Telegram or Discord, or managing communication workflows",
	})

	searchHandoff := agents.HandoffFromAgent(agents.HandoffFromAgentParams{
		Agent:                   searchAgent.Agent(),
		ToolNameOverride:        "handoff_to_search_agent",
		ToolDescriptionOverride: "Hand off to the Search Agent for web searches, fetching URL content, finding current information, or researching topics on the internet",
	})

	taskHandoff := agents.HandoffFromAgent(agents.HandoffFromAgentParams{
		Agent:                   taskAgent.Agent(),
		ToolNameOverride:        "handoff_to_task_agent",
		ToolDescriptionOverride: "Hand off to the Task Agent for managing tasks, creating to-dos, updating task status, or organizing task lists. Only hand off when specifically mentioning actions that involve tasks or to-dos",
	})

	// Get sysprompt path
	path := config.Get("OVERSEER_SYSPROMPT_PATH")
	if path == "" {
		return nil, errors.New("OVERSEER_SYSPROMPT_PATH not set in environment")
	}

	// Load instructions from file with fallback to hardcoded version
	instructions, err := utils.LoadPrompt(path)
	if err != nil {
		return nil, err
	}

	// Create the overseer agent with handoffs
	agentInstance := agents.New("overseer-agent").
		WithInstructions(instructions).
		WithModel(config.Get("MODEL")).
		WithHandoffs(
			memoryHandoff,
			communicationHandoff,
			searchHandoff,
			taskHandoff,
		)

	oa := &OverseerAgent{
		agent:        agentInstance,
		config:       config,
		memoryStore:  memoryStore,
		sessionStore: sessionStore,
	}

	return oa, nil
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
