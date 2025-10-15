package schedule

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethanbaker/assistant/internal/stores/memory"
	"github.com/ethanbaker/assistant/internal/stores/session"
	"github.com/ethanbaker/assistant/pkg/agent"
	"github.com/ethanbaker/assistant/pkg/utils"
	"github.com/nlpodyssey/openai-agents-go/agents"
)

// ScheduleAgent provides task management capabilities
type ScheduleAgent struct {
	agent           *agents.Agent
	config          *utils.Config
	memoryStore     *memory.Store
	sessionStore    session.Store
	basePrompt      string
	calendarService *CalendarService
	timezone        *time.Location
}

// NewScheduleAgent creates a new schedule agent
func NewScheduleAgent(memoryStore *memory.Store, sessionStore session.Store, config *utils.Config) (*ScheduleAgent, error) {
	var err error

	sa := &ScheduleAgent{
		config:       config,
		memoryStore:  memoryStore,
		sessionStore: sessionStore,
	}

	// Load timezone from config
	tz := config.Get("TIMEZONE")
	if tz == "" {
		return nil, errors.New("TIMEZONE not set in environment")
	}

	sa.timezone, err = time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("failed to load timezone %s: %w", tz, err)
	}

	// Get sysprompt path
	path := config.Get("SCHEDULE_SYSPROMPT_PATH")
	if path == "" {
		return nil, errors.New("SCHEDULE_SYSPROMPT_PATH not set in environment")
	}

	// Load instructions from file with fallback to hardcoded version
	sa.basePrompt, err = utils.LoadPrompt(path)
	if err != nil {
		return nil, err
	}

	// Initialize calendar service
	sa.calendarService, err = NewCalendarService(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize calendar service: %w", err)
	}

	// Create the underlying agent
	sa.agent = agents.New("schedule-agent").
		WithModel(config.Get("MODEL")).
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

// Config returns the agent configuration
func (sa *ScheduleAgent) Config() *utils.Config {
	return sa.config
}

// ShouldDryRun determines if the agent should run in dry-run mode
func (sa *ScheduleAgent) ShouldDryRun(ctx context.Context) bool {
	return sa.config.GetBool("DRY_RUN")
}

// getPrompt returns the prompt for the agent
func (sa *ScheduleAgent) getPrompt(ctx context.Context, a *agents.Agent) (string, error) {
	now := time.Now()

	builder := agent.NewPromptBuilder(sa.basePrompt)
	builder.AddContext("Current time: " + now.Format("15:04:05 MST"))
	builder.AddContext("Today's date: " + now.Format("2006-01-02"))

	// Format the next week's dates
	weekDates := "Following Week:\n"
	for i := range 7 {
		day := now.AddDate(0, 0, i)
		weekDates += "  - " + day.Format("Monday, 2006-01-02") + "\n"
	}
	builder.AddContext(weekDates)

	// Add user specific calendars
	calendars := "Calendars:\n"
	for _, cal := range sa.calendarService.calendarConfig.Calendars {
		calendars += fmt.Sprintf("  - **%s**: %s\n", cal.Name, cal.Description)
	}
	builder.AddContext(calendars)

	return builder.Build(), nil
}
