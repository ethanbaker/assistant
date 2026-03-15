package domain

import (
	"context"

	"github.com/nlpodyssey/openai-agents-go/agents"
)

// CustomAgent defines the interface for all custom agents in the system
type CustomAgent interface {
	// Agent returns the underlying openai-agents-go instance
	Agent() *agents.Agent

	// ID returns the unique identifier for this agent
	ID() string

	// ShouldDryRun determines if the agent should run tools with or without user interaction
	ShouldDryRun(ctx context.Context) bool
}
