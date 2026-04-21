package task

import (
	"context"
	"errors"
	"os"

	"github.com/ethanbaker/assistant/internal/prompts"
	"github.com/ethanbaker/assistant/internal/services/notion"
	"github.com/ethanbaker/assistant/pkg/config"
	"github.com/nlpodyssey/openai-agents-go/agents"
)

// TaskAgentConfig provided on construct
type TaskAgentConfig struct {
	Model             string
	PromptFile        string
	NotionTaskService notion.TaskService
}

// TaskAgent provides task management capabilities
type TaskAgent struct {
	agent      *agents.Agent
	basePrompt string

	taskService notion.TaskService
}

// NewTaskAgent creates a new task agent
func NewTaskAgent(cfg TaskAgentConfig) (*TaskAgent, error) {
	// Validate config
	if cfg.Model == "" {
		return nil, errors.New("Model is required")
	}
	if cfg.PromptFile == "" {
		return nil, errors.New("Prompt file path is required")
	}
	if cfg.NotionTaskService == nil {
		return nil, errors.New("NotionTaskService is required")
	}

	ta := &TaskAgent{
		taskService: cfg.NotionTaskService,
	}

	// Load instructions from file
	data, err := os.ReadFile(cfg.PromptFile)
	if err != nil {
		return nil, err
	}
	ta.basePrompt = string(data)

	// Create the underlying agent
	ta.agent = agents.New("task-agent").
		WithModel(cfg.Model).
		WithInstructionsFunc(ta.getPrompt)

	// Register tools
	ta.registerTools()

	return ta, nil
}

// Agent returns the underlying openai-agents-go instance
func (ta *TaskAgent) Agent() *agents.Agent {
	return ta.agent
}

// ID returns the agent identifier
func (ta *TaskAgent) ID() string {
	return "task-agent"
}

// ShouldDryRun determines if the agent should run in dry-run mode
func (ta *TaskAgent) ShouldDryRun(ctx context.Context) bool {
	return config.GetenvValue("DRY_RUN") == "true"
}

// getPrompt returns the prompt for the agent
func (ta *TaskAgent) getPrompt(ctx context.Context, a *agents.Agent) (string, error) {
	builder := prompts.NewPromptBuilder(ta.basePrompt)
	return builder.Build(), nil
}
