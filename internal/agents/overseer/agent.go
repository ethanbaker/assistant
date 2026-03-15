package overseer

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/ethanbaker/assistant/internal/config"
	"github.com/ethanbaker/assistant/internal/domain"
	"github.com/ethanbaker/assistant/internal/prompts"
	"github.com/nlpodyssey/openai-agents-go/agents"
)

// OverseerAgentConfig provided on construct
type OverseerAgentConfig struct {
	Model      string
	PromptFile string
	Handoffs   []domain.Handoff
}

// OverseerAgent coordinates and hands off to specialized agents
type OverseerAgent struct {
	agent      *agents.Agent
	basePrompt string
}

// NewOverseerAgent creates a new overseer agent with handoffs to all specialized agents
func NewOverseerAgent(cfg OverseerAgentConfig) (*OverseerAgent, error) {
	if cfg.Model == "" {
		return nil, errors.New("Model is required")
	}
	if cfg.PromptFile == "" {
		return nil, errors.New("PromptFile is required")
	}
	if len(cfg.Handoffs) == 0 {
		return nil, errors.New("Handoffs are required")
	}

	oa := &OverseerAgent{}

	data, err := os.ReadFile(cfg.PromptFile)
	if err != nil {
		return nil, err
	}
	oa.basePrompt = string(data)

	handoffs := make([]agents.Handoff, 0, len(cfg.Handoffs))
	for i, handoff := range cfg.Handoffs {
		if handoff.ToolName == "" {
			return nil, fmt.Errorf("Handoffs[%d].ToolName is required", i)
		}
		if handoff.ToolDescription == "" {
			return nil, fmt.Errorf("Handoffs[%d].ToolDescription is required", i)
		}
		if handoff.Agent == nil {
			return nil, fmt.Errorf("Handoffs[%d].Agent is required", i)
		}

		handoffs = append(handoffs, agents.HandoffFromAgent(agents.HandoffFromAgentParams{
			Agent:                   handoff.Agent.Agent(),
			ToolNameOverride:        handoff.ToolName,
			ToolDescriptionOverride: handoff.ToolDescription,
		}))
	}

	// Create the overseer agent with handoffs
	oa.agent = agents.New("overseer-agent").
		WithModel(cfg.Model).
		WithInstructionsFunc(oa.getPrompt).
		WithHandoffs(handoffs...)

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

// ShouldDryRun determines if the agent should run in dry-run mode
func (oa *OverseerAgent) ShouldDryRun(ctx context.Context) bool {
	return config.GetenvValue("DRY_RUN") == "true"
}

// getPrompt returns the prompt for the agent
func (oa *OverseerAgent) getPrompt(ctx context.Context, a *agents.Agent) (string, error) {
	builder := prompts.NewPromptBuilder(oa.basePrompt)
	return builder.Build(), nil
}
