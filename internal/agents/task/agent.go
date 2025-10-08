package task

import (
	"context"
	"errors"
	"net/http"
	"time"

	notionapi "github.com/dstotijn/go-notion"
	"github.com/ethanbaker/assistant/internal/stores/memory"
	"github.com/ethanbaker/assistant/internal/stores/session"
	"github.com/ethanbaker/assistant/pkg/agent"
	"github.com/ethanbaker/assistant/pkg/utils"
	"github.com/nlpodyssey/openai-agents-go/agents"
)

// TaskAgent provides task management capabilities
type TaskAgent struct {
	agent        *agents.Agent
	config       *utils.Config
	memoryStore  *memory.Store
	sessionStore session.Store
	basePrompt   string
	notionClient *notionapi.Client
}

// NewTaskAgent creates a new task agent
func NewTaskAgent(memoryStore *memory.Store, sessionStore session.Store, config *utils.Config) (*TaskAgent, error) {
	ta := &TaskAgent{
		config:       config,
		memoryStore:  memoryStore,
		sessionStore: sessionStore,
	}

	// Get sysprompt path
	path := config.Get("TASK_SYSPROMPT_PATH")
	if path == "" {
		return nil, errors.New("TASK_SYSPROMPT_PATH not set in environment")
	}

	// Load instructions from file with fallback to hardcoded version
	var err error
	ta.basePrompt, err = utils.LoadPrompt(path)
	if err != nil {
		return nil, err
	}

	// Initialize Notion client
	token := config.Get("NOTION_API_TOKEN")
	if token == "" {
		return nil, errors.New("NOTION_API_TOKEN not set in environment")
	}
	ta.notionClient = notionapi.NewClient(token, notionapi.WithHTTPClient(&http.Client{
		Timeout: 20 * time.Second,
	}))

	// Create the underlying agent
	ta.agent = agents.New("task-agent").
		WithModel(config.Get("MODEL"))

	// Register tools
	ta.registerTools()

	return ta, nil
}

// Agent returns the underlying openai-agents-go instance
func (ta *TaskAgent) Agent() *agents.Agent {
	now := time.Now()

	builder := agent.NewPromptBuilder(ta.basePrompt)
	builder.AddContext("Time: " + now.Format("15:04:05 MST"))
	builder.AddContext("Date: " + now.Format("2006-01-02"))

	// Format the next week's dates
	weekDates := "Next Week's Dates:\n"
	for i := 0; i < 7; i++ {
		day := now.AddDate(0, 0, i)
		weekDates += "  - " + day.Format("Monday, 2006-01-02") + "\n"
	}

	return ta.agent.WithInstructions(builder.Build())
}

// ID returns the agent identifier
func (ta *TaskAgent) ID() string {
	return "task-agent"
}

// Config returns the agent configuration
func (ta *TaskAgent) Config() *utils.Config {
	return ta.config
}

// ShouldDryRun determines if the agent should run in dry-run mode
func (ta *TaskAgent) ShouldDryRun(ctx context.Context) bool {
	return ta.config.GetBool("DRY_RUN")
}

// getNotionClient returns the Notion client instance
func (ta *TaskAgent) getNotionClient() *notionapi.Client {
	return ta.notionClient
}
