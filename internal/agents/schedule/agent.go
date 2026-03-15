package schedule

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"

	"github.com/ethanbaker/assistant/internal/config"
	"github.com/ethanbaker/assistant/internal/prompts"
	"github.com/ethanbaker/assistant/internal/services/gcal"
	"github.com/nlpodyssey/openai-agents-go/agents"
)

// ScheduleAgentConfig provided on construct
type ScheduleAgentConfig struct {
	Model           string
	PromptFile      string
	CalendarService gcal.CalendarService
}

// ScheduleAgent provides task management capabilities
type ScheduleAgent struct {
	agent      *agents.Agent
	basePrompt string

	calendarService gcal.CalendarService
}

// NewScheduleAgent creates a new schedule agent
func NewScheduleAgent(cfg ScheduleAgentConfig) (*ScheduleAgent, error) {
	// Validate config
	if cfg.Model == "" {
		return nil, errors.New("Model is required")
	}
	if cfg.PromptFile == "" {
		return nil, errors.New("PromptFile is required")
	}
	if cfg.CalendarService == nil || reflect.ValueOf(cfg.CalendarService).IsNil() {
		return nil, errors.New("CalendarService is required")
	}

	sa := &ScheduleAgent{
		calendarService: cfg.CalendarService,
	}

	// Load instructions from file
	data, err := os.ReadFile(cfg.PromptFile)
	if err != nil {
		return nil, err
	}
	sa.basePrompt = string(data)

	// Create the underlying agent
	sa.agent = agents.New("schedule-agent").
		WithModel(cfg.Model).
		WithInstructionsFunc(sa.getPrompt)

	// Register tools
	sa.registerTools()

	return sa, nil
}

// Agent returns the underlying openai-agents-go instance
func (sa *ScheduleAgent) Agent() *agents.Agent {
	return sa.agent
}

// ID returns the agent identifier
func (sa *ScheduleAgent) ID() string {
	return "schedule-agent"
}

// ShouldDryRun determines if the agent should run in dry-run mode
func (sa *ScheduleAgent) ShouldDryRun(ctx context.Context) bool {
	return config.GetenvValue("DRY_RUN") == "true"
}

// getPrompt returns the prompt for the agent
func (sa *ScheduleAgent) getPrompt(ctx context.Context, a *agents.Agent) (string, error) {
	builder := prompts.NewPromptBuilder(sa.basePrompt)

	// Add user specific calendars
	calendars := "Calendars:\n"
	for _, cal := range sa.calendarService.GetCalendars() {
		calendars += fmt.Sprintf("  - **%s**: %s\n", cal.Name, cal.Description)
	}
	builder.AddContext(calendars)

	return builder.Build(), nil
}
