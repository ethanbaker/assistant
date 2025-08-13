// tools.go handles registering tools for the MemoryAgent
package memoryagent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nlpodyssey/openai-agents-go/agents"
)

// registerTools registers the memory-related tools
func (ma *MemoryAgent) registerTools() {
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
			"required": []string{"query"},
		},
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return ma.handleSearchSessions(ctx, arguments)
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
			"required": []string{"key"},
		},
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return ma.handleGetFact(ctx, arguments)
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
			"required": []string{"key", "value"},
		},
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return ma.handleSetFact(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}

	// List facts tool
	listFactsTool := agents.FunctionTool{
		Name:        "list_facts",
		Description: "List all stored facts",
		ParamsJSONSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return ma.handleListFacts(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}

	// Add tools to agent
	ma.agent.Tools = []agents.Tool{
		searchTool,
		getFactTool,
		setFactTool,
		listFactsTool,
	}
}

// handleSearchSessions handles session transcript searches
func (ma *MemoryAgent) handleSearchSessions(ctx context.Context, arguments string) (string, error) {
	if ma.ShouldDryRun(ctx) {
		return fmt.Sprintf("DRY RUN: Would search sessions with query: %s", arguments), nil
	}

	// Unmarshal the arguments
	var args struct {
		Query string `json:"query"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Search session transcripts with the provided query
	transcripts, err := ma.sessionStore.SearchSessionTranscripts(ctx, args.Query)
	if err != nil {
		return "", fmt.Errorf("failed to search sessions: %w", err)
	}

	if len(transcripts) == 0 {
		return "No relevant conversations found.", nil
	}

	// Format the results
	result := fmt.Sprintf("Found %d relevant conversation segments:\n", len(transcripts))
	for i, transcript := range transcripts {
		if i >= 5 { // Limit to top 5 results
			break
		}
		result += fmt.Sprintf("\n%d. [%s] %s", i+1, transcript.CreatedAt.Format("2006-01-02 15:04"), transcript.Data[:min(200, len(transcript.Data))])
		if len(transcript.Data) > 200 {
			result += "..."
		}
	}

	return result, nil
}

// handleGetFact handles fact retrieval
func (ma *MemoryAgent) handleGetFact(ctx context.Context, arguments string) (string, error) {
	if ma.ShouldDryRun(ctx) {
		return fmt.Sprintf("DRY RUN: Would get fact with arguments: %s", arguments), nil
	}

	// Unmarshal the arguments
	var args struct {
		Key string `json:"key"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Retrieve the fact from memory store
	fact, err := ma.memoryStore.GetFact(ctx, args.Key)
	if err != nil {
		return "", fmt.Errorf("failed to get fact: %w", err)
	}

	// Check if the fact exists
	if fact == nil {
		return fmt.Sprintf("No fact found for key: %s", args.Key), nil
	}

	// Format the result
	return fmt.Sprintf("Fact '%s': %s", fact.Key, fact.Value), nil
}

// handleSetFact handles fact storage
func (ma *MemoryAgent) handleSetFact(ctx context.Context, arguments string) (string, error) {
	if ma.ShouldDryRun(ctx) {
		return fmt.Sprintf("DRY RUN: Would set fact with arguments: %s", arguments), nil
	}

	// Unmarshal the arguments
	var args struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Store the fact in memory store
	if err := ma.memoryStore.SetFact(ctx, args.Key, args.Value); err != nil {
		return "", fmt.Errorf("failed to set fact: %w", err)
	}

	return fmt.Sprintf("Successfully stored fact '%s': %s", args.Key, args.Value), nil
}

// handleListFacts handles listing all facts
func (ma *MemoryAgent) handleListFacts(ctx context.Context, _ string) (string, error) {
	if ma.ShouldDryRun(ctx) {
		return "DRY RUN: Would list all facts", nil
	}

	// List all facts from memory store
	facts, err := ma.memoryStore.ListAllFacts(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list facts: %w", err)
	}

	if len(facts) == 0 {
		return "No facts stored yet.", nil
	}

	// Format the results
	result := fmt.Sprintf("Stored facts (%d total):\n", len(facts))
	for _, fact := range facts {
		result += fmt.Sprintf("- %s: %s\n", fact.Key, fact.Value)
	}

	return result, nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
