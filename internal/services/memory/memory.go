package memory

import (
	"context"
	"errors"

	"github.com/ethanbaker/assistant/internal/domain"
	"github.com/ethanbaker/assistant/internal/repositories/keyfact"
	"github.com/ethanbaker/assistant/internal/repositories/session"
	"github.com/nlpodyssey/openai-agents-go/agents"
	"github.com/openai/openai-go/v2/packages/param"
)

// MemoryToolRegistryConfig provided upon initialization
type MemoryToolRegistryConfig struct {
	FactRepository    keyfact.Repository
	SessionRepository session.Repository
}

// MemoryToolRegistry contains methods to register memory tools to agents
type MemoryToolRegistry struct {
	factRepository    keyfact.Repository
	sessionRepository session.Repository
}

// NewMemoryToolRegistry creates a new memory tool register
func NewMemoryToolRegistry(cfg MemoryToolRegistryConfig) (*MemoryToolRegistry, error) {
	if cfg.FactRepository == nil {
		return nil, errors.New("FactRepository is required")
	}
	if cfg.SessionRepository == nil {
		return nil, errors.New("SessionRepository is required")
	}

	return &MemoryToolRegistry{
		factRepository:    cfg.FactRepository,
		sessionRepository: cfg.SessionRepository,
	}, nil
}

// RegisterTools registers the memory-related tools
func (r *MemoryToolRegistry) RegisterTools(customAgents ...domain.CustomAgent) {
	// Search session transcripts tool
	searchTool := agents.FunctionTool{
		Name:        "search_sessions",
		Description: "Search through past conversation transcripts",
		ParamsJSONSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "Search query to find relevant conversations",
				},
			},
			"additionalProperties": false,
			"required":             []string{"query"},
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return r.handleSearchSessions(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}

	// Get fact tool
	getFactTool := agents.FunctionTool{
		Name:        "get_fact",
		Description: "Retrieve a stored fact by key",
		ParamsJSONSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"key": map[string]any{
					"type":        "string",
					"description": "The key of the fact to retrieve",
				},
			},
			"additionalProperties": false,
			"required":             []string{"key"},
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return r.handleGetFact(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}

	// Set fact tool
	setFactTool := agents.FunctionTool{
		Name:        "set_fact",
		Description: "Store or update a fact",
		ParamsJSONSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"key": map[string]any{
					"type":        "string",
					"description": "The key for the fact",
				},
				"value": map[string]any{
					"type":        "string",
					"description": "The value of the fact",
				},
			},
			"additionalProperties": false,
			"required":             []string{"key", "value"},
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return r.handleSetFact(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}

	// List facts tool
	listFactsTool := agents.FunctionTool{
		Name:        "list_facts",
		Description: "List all stored facts",
		ParamsJSONSchema: map[string]any{
			"type":                 "object",
			"properties":           map[string]any{},
			"additionalProperties": false,
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return r.handleListFacts(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}

	// Add tools to agent
	for _, a := range customAgents {
		a.Agent().Tools = append(a.Agent().Tools, []agents.Tool{
			searchTool,
			getFactTool,
			setFactTool,
			listFactsTool,
		}...)
	}
}
