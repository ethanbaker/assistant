package agent

import (
	"fmt"
	"strings"
)

// PromptBuilder helps construct dynamic prompts for agents
type PromptBuilder struct {
	systemPrompt string
	context      []string
	facts        map[string]string
}

// NewPromptBuilder creates a new prompt builder with a base system prompt
func NewPromptBuilder(systemPrompt string) *PromptBuilder {
	return &PromptBuilder{
		systemPrompt: systemPrompt,
		context:      make([]string, 0),
		facts:        make(map[string]string),
	}
}

// AddContext adds contextual information to the prompt
func (pb *PromptBuilder) AddContext(context string) *PromptBuilder {
	pb.context = append(pb.context, context)
	return pb
}

// AddFact adds a key-value fact to the prompt
func (pb *PromptBuilder) AddFact(key, value string) *PromptBuilder {
	pb.facts[key] = value
	return pb
}

// Build constructs the final prompt
func (pb *PromptBuilder) Build() string {
	var parts []string

	// Start with system prompt
	parts = append(parts, pb.systemPrompt)

	// Add facts if any
	if len(pb.facts) > 0 {
		parts = append(parts, "\n## Key Facts:")
		for key, value := range pb.facts {
			parts = append(parts, fmt.Sprintf("- %s: %s", key, value))
		}
	}

	// Add context if any
	if len(pb.context) > 0 {
		parts = append(parts, "\n## Recent Context:")
		for _, ctx := range pb.context {
			parts = append(parts, fmt.Sprintf("- %s", ctx))
		}
	}

	return strings.Join(parts, "\n")
}
