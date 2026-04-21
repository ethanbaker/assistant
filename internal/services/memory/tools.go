package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// handleSearchSessions handles session transcript searches
func (r *MemoryToolRegistry) handleSearchSessions(ctx context.Context, arguments string) (string, error) {
	// Unmarshal the arguments
	var args struct {
		Query string `json:"query"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Search session transcripts with the provided query
	transcripts, err := r.sessionRepository.SearchSessionTranscripts(ctx, args.Query)
	if err != nil {
		return "", fmt.Errorf("failed to search sessions: %w", err)
	}

	if len(transcripts) == 0 {
		return "No relevant conversations found.", nil
	}

	// Format the results
	var result strings.Builder
	fmt.Fprintf(&result, "Found %d relevant conversation segments:\n", len(transcripts))
	for i, transcript := range transcripts {
		if i >= 5 { // Limit to top 5 results
			break
		}

		fmt.Fprintf(&result, "\n%d. [%s] %s", i+1, transcript.CreatedAt.Format("2006-01-02 15:04"), transcript.Data[:min(200, len(transcript.Data))])
		if len(transcript.Data) > 200 {
			result.WriteString("...")
		}
	}

	return result.String(), nil
}

// handleGetFact handles fact retrieval
func (r *MemoryToolRegistry) handleGetFact(ctx context.Context, arguments string) (string, error) {
	// Unmarshal the arguments
	var args struct {
		Key string `json:"key"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Retrieve the fact from memory store
	fact, err := r.factRepository.GetFact(ctx, args.Key)
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
func (r *MemoryToolRegistry) handleSetFact(ctx context.Context, arguments string) (string, error) {
	// Unmarshal the arguments
	var args struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Store the fact in memory store
	if err := r.factRepository.SetFact(ctx, args.Key, args.Value); err != nil {
		return "", fmt.Errorf("failed to set fact: %w", err)
	}

	return fmt.Sprintf("Successfully stored fact '%s': %s", args.Key, args.Value), nil
}

// handleListFacts handles listing all facts
func (r *MemoryToolRegistry) handleListFacts(ctx context.Context, _ string) (string, error) {
	// List all facts from memory store
	facts, err := r.factRepository.ListAllFacts(ctx)
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
