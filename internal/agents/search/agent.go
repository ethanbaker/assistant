package search

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/ethanbaker/assistant/internal/stores/memory"
	"github.com/ethanbaker/assistant/internal/stores/session"
	"github.com/ethanbaker/assistant/pkg/utils"
	"github.com/nlpodyssey/openai-agents-go/agents"
)

// SearchResult represents a single search result
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
	Engine  string `json:"engine"`
}

// SearXNGResponse represents the response from SearXNG API
type SearXNGResponse struct {
	Query           string         `json:"query"`
	NumberOfResults int            `json:"number_of_results"`
	Results         []SearchResult `json:"results"`
	Suggestions     []string       `json:"suggestions"`
	Infoboxes       []any          `json:"infoboxes"`
}

// SearchAgent provides internet search and information gathering capabilities
type SearchAgent struct {
	agent        *agents.Agent
	config       *utils.Config
	memoryStore  *memory.Store
	sessionStore *session.Store
	searxngURL   string
	httpClient   *http.Client
}

// NewSearchAgent creates a new search agent
func NewSearchAgent(memoryStore *memory.Store, sessionStore *session.Store, config *utils.Config) (*SearchAgent, error) {
	// Get SearXNG URL from environment, default to localhost
	searxngURL := os.Getenv("SEARXNG_URL")
	if searxngURL == "" {
		return nil, errors.New("SEARXNG_URL not set in environment")
	}

	sa := &SearchAgent{
		config:       config,
		memoryStore:  memoryStore,
		sessionStore: sessionStore,
		searxngURL:   searxngURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Get sysprompt path
	path := config.Get("SEARCH_SYSPROMPT_PATH")
	if path == "" {
		return nil, errors.New("SEARCH_SYSPROMPT_PATH not set in environment")
	}

	// Load instructions from file
	instructions, err := utils.LoadPrompt(path)
	if err != nil {
		return nil, err
	}

	// Create the underlying agent
	agentInstance := agents.New("search-agent").
		WithInstructions(instructions).
		WithModel(config.Get("MODEL"))

	sa.agent = agentInstance

	// Register tools
	sa.registerTools()

	return sa, nil
}

// Agent returns the underlying openai-agents-go instance
func (sa *SearchAgent) Agent() *agents.Agent {
	return sa.agent
}

// ID returns the agent identifier
func (sa *SearchAgent) ID() string {
	return "search-agent"
}

// Config returns the agent configuration
func (sa *SearchAgent) Config() *utils.Config {
	return sa.config
}

// ShouldDryRun determines if the agent should run in dry-run mode
func (sa *SearchAgent) ShouldDryRun(ctx context.Context) bool {
	return sa.config.GetBool("DRY_RUN")
}
