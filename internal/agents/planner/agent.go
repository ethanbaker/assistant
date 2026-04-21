package planner

import (
	"context"
	"errors"
	"os"

	"github.com/ethanbaker/assistant/internal/domain"
	"github.com/ethanbaker/assistant/internal/prompts"
	"github.com/ethanbaker/assistant/pkg/config"
	"github.com/nlpodyssey/openai-agents-go/agents"
	"github.com/nlpodyssey/openai-agents-go/modelsettings"
	"github.com/openai/openai-go/v2/packages/param"
)

// PlannerAgentConfig provided on construct
type PlannerAgentConfig struct {
	Model      string
	PromptFile string
	Overseer   domain.CustomAgent
}

// PlannerAgent decomposes complex requests into a structured JSON plan artifact.
type PlannerAgent struct {
	agent      *agents.Agent
	basePrompt string
}

// NewPlannerAgent creates a new planner agent.
func NewPlannerAgent(cfg PlannerAgentConfig) (*PlannerAgent, error) {
	if cfg.Model == "" {
		return nil, errors.New("Model is required")
	}
	if cfg.PromptFile == "" {
		return nil, errors.New("PromptFile is required")
	}
	if cfg.Overseer == nil {
		return nil, errors.New("Overseer is required")
	}

	pa := &PlannerAgent{}

	data, err := os.ReadFile(cfg.PromptFile)
	if err != nil {
		return nil, err
	}
	pa.basePrompt = string(data)

	handoff := agents.HandoffFromAgent(agents.HandoffFromAgentParams{
		Agent:                   cfg.Overseer.Agent(),
		ToolNameOverride:        "handoff_to_overseer",
		ToolDescriptionOverride: "",
	})

	pa.agent = agents.New("planner-agent").
		WithModel(cfg.Model).
		WithModelSettings(modelsettings.ModelSettings{
			Temperature: param.NewOpt(0.2),
			MaxTokens:   param.NewOpt[int64](1024),
		}).
		WithInstructionsFunc(pa.getPrompt).
		WithHandoffs(handoff)

	return pa, nil
}

// Agent returns the underlying openai-agents-go instance.
func (pa *PlannerAgent) Agent() *agents.Agent {
	return pa.agent
}

// ID returns the agent identifier.
func (pa *PlannerAgent) ID() string {
	return "planner-agent"
}

// ShouldDryRun determines if the agent should run in dry-run mode.
func (pa *PlannerAgent) ShouldDryRun(ctx context.Context) bool {
	return config.GetenvValue("DRY_RUN") == "true"
}

// getPrompt returns the prompt for the agent.
func (pa *PlannerAgent) getPrompt(ctx context.Context, a *agents.Agent) (string, error) {
	builder := prompts.NewPromptBuilder(pa.basePrompt)
	return builder.Build(), nil
}
