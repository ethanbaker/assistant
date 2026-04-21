package entry

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/ethanbaker/assistant/internal/domain"
	"github.com/ethanbaker/assistant/internal/prompts"
	"github.com/ethanbaker/assistant/pkg/config"
	"github.com/nlpodyssey/openai-agents-go/agents"
	"github.com/nlpodyssey/openai-agents-go/modelsettings"
	"github.com/openai/openai-go/v2/packages/param"
)

// EntryAgentConfig provided on construct
type EntryAgentConfig struct {
	Model      string
	PromptFile string
	Handoffs   []domain.Handoff
}

// EntryAgent is the first agent to receive every user request.
// It classifies the request and routes to the appropriate downstream agent
// via handoffs. It performs no task execution itself.
type EntryAgent struct {
	agent      *agents.Agent
	basePrompt string
}

// NewEntryAgent creates a new entry agent with handoffs to downstream agents.
func NewEntryAgent(cfg EntryAgentConfig) (*EntryAgent, error) {
	if cfg.Model == "" {
		return nil, errors.New("Model is required")
	}
	if cfg.PromptFile == "" {
		return nil, errors.New("PromptFile is required")
	}
	if len(cfg.Handoffs) == 0 {
		return nil, errors.New("Handoffs are required")
	}

	ea := &EntryAgent{}

	data, err := os.ReadFile(cfg.PromptFile)
	if err != nil {
		return nil, err
	}
	ea.basePrompt = string(data)

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

	ea.agent = agents.New("entry-agent").
		WithModel(cfg.Model).
		WithModelSettings(modelsettings.ModelSettings{
			Temperature: param.NewOpt(0.0),
			MaxTokens:   param.NewOpt[int64](256),
		}).
		WithInstructionsFunc(ea.getPrompt).
		WithHandoffs(handoffs...)

	return ea, nil
}

// Agent returns the underlying openai-agents-go instance.
func (ea *EntryAgent) Agent() *agents.Agent {
	return ea.agent
}

// ID returns the agent identifier.
func (ea *EntryAgent) ID() string {
	return "entry-agent"
}

// ShouldDryRun determines if the agent should run in dry-run mode.
func (ea *EntryAgent) ShouldDryRun(ctx context.Context) bool {
	return config.GetenvValue("DRY_RUN") == "true"
}

// getPrompt returns the prompt for the agent
func (ea *EntryAgent) getPrompt(ctx context.Context, a *agents.Agent) (string, error) {
	builder := prompts.NewPromptBuilder(ea.basePrompt)
	return builder.Build(), nil
}
