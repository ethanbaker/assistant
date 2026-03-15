package searxng

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/ethanbaker/assistant/internal/domain"
	"github.com/nlpodyssey/openai-agents-go/agents"
	"github.com/openai/openai-go/v2/packages/param"
)

// Limit content length to prevent overwhelming the model
const maxContentLength = 10000

// SearxngToolRegisterConfig provided upon initialization
type SearxngToolRegisterConfig struct {
	SearxngUrl string
}

// SearxngToolRegister contains methods to register search tools to agents
type SearxngToolRegister struct {
	searxngUrl string
	httpClient *http.Client
}

// NewSearxngToolRegister creates a new search tool register
func NewSearxngToolRegister(cfg SearxngToolRegisterConfig) (*SearxngToolRegister, error) {
	if cfg.SearxngUrl == "" {
		return nil, errors.New("SearxngUrl is required")
	}

	return &SearxngToolRegister{
		searxngUrl: cfg.SearxngUrl,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// RegisterTools registers the search-related tools
func (r *SearxngToolRegister) RegisterTools(customAgents ...domain.CustomAgent) {
	// Web search tool
	webSearchTool := agents.FunctionTool{
		Name:        "web_search",
		Description: "Search the internet for information using a meta-search engine",
		ParamsJSONSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "The search query to execute",
				},
				"num_results": map[string]any{
					"type":        "integer",
					"description": "Maximum number of results to return (optional, defaults to 10)",
					"minimum":     1,
					"maximum":     50,
					"default":     10,
				},
				"category": map[string]any{
					"type":        "string",
					"description": "Search category: 'general', 'news', 'images', or 'map' (optional, defaults to 'general')",
					"enum":        []string{"general", "news", "images", "map"},
					"default":     "general",
				},
			},
			"additionalProperties": false,
			"required":             []string{"query", "num_results", "category"},
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return r.handleWebSearch(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}

	// URL fetch tool
	fetchURLTool := agents.FunctionTool{
		Name:        "fetch_url",
		Description: "Fetch and extract text content from a specific URL",
		ParamsJSONSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url": map[string]any{
					"type":        "string",
					"description": "The URL to fetch content from",
				},
				"extract_main_content": map[string]any{
					"type":        "boolean",
					"description": "Whether to extract only main content (removes ads, navigation, etc.) or full HTML text (optional, defaults to true)",
					"default":     true,
				},
			},
			"additionalProperties": false,
			"required":             []string{"url", "extract_main_content"},
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return r.handleFetchURL(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}

	// Summarize search results tool
	summarizeResultsTool := agents.FunctionTool{
		Name:        "summarize_search_results",
		Description: "Summarize and synthesize information from multiple search results",
		ParamsJSONSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"results": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"url": map[string]any{
								"type": "string",
							},
							"title": map[string]any{
								"type": "string",
							},
							"content": map[string]any{
								"type": "string",
							},
						},
						"additionalProperties": false,
						"required":             []string{"url", "title", "content"},
					},
					"description": "Array of search results to summarize",
				},
				"focus_query": map[string]any{
					"type":        "string",
					"description": "Specific question or topic to focus the summary on (optional)",
					"default":     "",
				},
				"summary_style": map[string]any{
					"type":        "string",
					"description": "Summary style: 'brief', 'detailed', or 'bullet_points' (optional, defaults to 'detailed')",
					"enum":        []string{"brief", "detailed", "bullet_points"},
					"default":     "detailed",
				},
			},
			"additionalProperties": false,
			"required":             []string{"results", "focus_query", "summary_style"},
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return r.handleSummarizeResults(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}

	// Register all tools with the agent
	for _, a := range customAgents {
		a.Agent().Tools = append(a.Agent().Tools, []agents.Tool{
			webSearchTool,
			fetchURLTool,
			summarizeResultsTool,
		}...)
	}
}
