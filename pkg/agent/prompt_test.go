package agent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test cases for the PromptBuilder functionality
func TestPromptBuilder_Creation(t *testing.T) {
	tests := []struct {
		name         string
		systemPrompt string
		want         string
	}{
		{
			name:         "simple system prompt",
			systemPrompt: "You are a helpful assistant.",
			want:         "You are a helpful assistant.",
		},
		{
			name:         "empty system prompt",
			systemPrompt: "",
			want:         "",
		},
		{
			name:         "multiline system prompt",
			systemPrompt: "You are a helpful assistant.\nYou provide accurate information.",
			want:         "You are a helpful assistant.\nYou provide accurate information.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewPromptBuilder(tt.systemPrompt)

			require.NotNil(t, pb)

			result := pb.Build()
			assert.Equal(t, tt.want, result)
		})
	}
}

// Test cases for adding context to the prompt
func TestPromptBuilder_AddContext(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		contexts []string
		want     string
	}{
		{
			name:     "single context",
			base:     "Base prompt",
			contexts: []string{"User logged in"},
			want:     "Base prompt\n\n## Recent Context:\n- User logged in",
		},
		{
			name:     "multiple contexts",
			base:     "Base prompt",
			contexts: []string{"User logged in", "Session started", "Data loaded"},
			want:     "Base prompt\n\n## Recent Context:\n- User logged in\n- Session started\n- Data loaded",
		},
		{
			name:     "empty context",
			base:     "Base prompt",
			contexts: []string{""},
			want:     "Base prompt\n\n## Recent Context:\n- ",
		},
		{
			name:     "no contexts",
			base:     "Base prompt",
			contexts: []string{},
			want:     "Base prompt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewPromptBuilder(tt.base)

			for _, ctx := range tt.contexts {
				pb.AddContext(ctx)
			}

			result := pb.Build()
			assert.Equal(t, tt.want, result)
		})
	}
}

// Test cases for adding facts to the prompt
func TestPromptBuilder_AddFact(t *testing.T) {
	tests := []struct {
		name  string
		base  string
		facts map[string]string
		want  string
	}{
		{
			name: "single fact",
			base: "Base prompt",
			facts: map[string]string{
				"user": "alice",
			},
			want: "Base prompt\n\n## Key Facts:\n- user: alice",
		},
		{
			name: "multiple facts",
			base: "Base prompt",
			facts: map[string]string{
				"user":    "alice",
				"role":    "admin",
				"session": "12345",
			},
			want: "Base prompt\n\n## Key Facts:\n- user: alice\n- role: admin\n- session: 12345",
		},
		{
			name: "empty fact value",
			base: "Base prompt",
			facts: map[string]string{
				"empty": "",
			},
			want: "Base prompt\n\n## Key Facts:\n- empty: ",
		},
		{
			name:  "no facts",
			base:  "Base prompt",
			facts: map[string]string{},
			want:  "Base prompt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewPromptBuilder(tt.base)

			for key, value := range tt.facts {
				pb.AddFact(key, value)
			}

			result := pb.Build()

			// For multiple facts, the order might vary, so we need to check differently
			if len(tt.facts) > 1 {
				// Check that all expected parts are present
				assert.Contains(t, result, tt.base)
				assert.Contains(t, result, "## Key Facts:")
				for key, value := range tt.facts {
					expected := "- " + key + ": " + value
					assert.Contains(t, result, expected)
				}
			} else {
				// For single or no facts, exact match is fine
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

// Test cases for chaining adding methods in PromptBuilder
func TestPromptBuilder_ChainedMethods(t *testing.T) {
	pb := NewPromptBuilder("System prompt")

	// Test method chaining
	result := pb.
		AddContext("Context 1").
		AddFact("key1", "value1").
		AddContext("Context 2").
		AddFact("key2", "value2")

	assert.Equal(t, pb, result, "AddContext and AddFact should return the same PromptBuilder instance for chaining")

	finalPrompt := pb.Build()

	// Verify all components are present
	expectedComponents := []string{
		"System prompt",
		"## Key Facts:",
		"- key1: value1",
		"- key2: value2",
		"## Recent Context:",
		"- Context 1",
		"- Context 2",
	}

	for _, component := range expectedComponents {
		assert.Contains(t, finalPrompt, component, "Final prompt missing component %q. Got: %q", component, finalPrompt)
	}
}

// Test cases for complex scenarios in PromptBuilder
func TestPromptBuilder_ComplexScenarios(t *testing.T) {
	tests := []struct {
		name         string
		systemPrompt string
		contexts     []string
		facts        map[string]string
		validateFunc func(string) bool
		description  string
	}{
		{
			name:         "complete prompt with all sections",
			systemPrompt: "You are an AI assistant specializing in code review.",
			contexts:     []string{"Pull request opened", "CI/CD pipeline running", "Tests passing"},
			facts:        map[string]string{"repo": "assistant", "branch": "feature/testing", "author": "developer"},
			validateFunc: func(prompt string) bool {
				return strings.Contains(prompt, "AI assistant") &&
					strings.Contains(prompt, "## Key Facts:") &&
					strings.Contains(prompt, "## Recent Context:") &&
					strings.Contains(prompt, "repo: assistant") &&
					strings.Contains(prompt, "Pull request opened")
			},
			description: "should contain all sections in proper format",
		},
		{
			name:         "context only prompt",
			systemPrompt: "Simple prompt",
			contexts:     []string{"Event A", "Event B"},
			facts:        map[string]string{},
			validateFunc: func(prompt string) bool {
				return strings.Contains(prompt, "Simple prompt") &&
					strings.Contains(prompt, "## Recent Context:") &&
					!strings.Contains(prompt, "## Key Facts:") &&
					strings.Contains(prompt, "Event A")
			},
			description: "should contain only system prompt and context sections",
		},
		{
			name:         "facts only prompt",
			systemPrompt: "Another prompt",
			contexts:     []string{},
			facts:        map[string]string{"status": "active", "mode": "production"},
			validateFunc: func(prompt string) bool {
				return strings.Contains(prompt, "Another prompt") &&
					!strings.Contains(prompt, "## Recent Context:") &&
					strings.Contains(prompt, "## Key Facts:") &&
					strings.Contains(prompt, "status: active")
			},
			description: "should contain only system prompt and facts sections",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewPromptBuilder(tt.systemPrompt)

			for _, ctx := range tt.contexts {
				pb.AddContext(ctx)
			}

			for key, value := range tt.facts {
				pb.AddFact(key, value)
			}

			result := pb.Build()

			assert.True(t, tt.validateFunc(result), "Validation failed: %s. Got prompt: %q", tt.description, result)
		})
	}
}

// Test edge cases in PromptBuilder
func TestPromptBuilder_EdgeCases(t *testing.T) {
	t.Run("special characters in facts", func(t *testing.T) {
		pb := NewPromptBuilder("Base")
		pb.AddFact("json", `{"key": "value", "number": 42}`)
		pb.AddFact("special", "!@#$%^&*()_+-=[]{}|;:,.<>?")

		result := pb.Build()
		assert.Contains(t, result, `{"key": "value", "number": 42}`)
		assert.Contains(t, result, "!@#$%^&*()_+-=[]{}|;:,.<>?")
	})

	t.Run("newlines in context", func(t *testing.T) {
		pb := NewPromptBuilder("Base")
		pb.AddContext("Line 1\nLine 2\nLine 3")

		result := pb.Build()
		assert.Contains(t, result, "Line 1\nLine 2\nLine 3")
	})

	t.Run("overwriting facts", func(t *testing.T) {
		pb := NewPromptBuilder("Base")
		pb.AddFact("key", "value1")
		pb.AddFact("key", "value2") // Should overwrite

		result := pb.Build()
		assert.NotContains(t, result, "value1")
		assert.Contains(t, result, "value2")
	})

	t.Run("empty system prompt with content", func(t *testing.T) {
		pb := NewPromptBuilder("")
		pb.AddContext("Some context")
		pb.AddFact("key", "value")

		result := pb.Build()

		// Should still build properly even with empty system prompt
		assert.Contains(t, result, "## Key Facts:")
		assert.Contains(t, result, "## Recent Context:")
	})
}

// Test section ordering in PromptBuilder
func TestPromptBuilder_SectionOrdering(t *testing.T) {
	pb := NewPromptBuilder("System message")
	pb.AddFact("fact", "value")
	pb.AddContext("context")

	result := pb.Build()

	// Verify the order: System prompt -> Facts -> Context
	systemIndex := strings.Index(result, "System message")
	factsIndex := strings.Index(result, "## Key Facts:")
	contextIndex := strings.Index(result, "## Recent Context:")

	require.True(t, systemIndex != -1 && factsIndex != -1 && contextIndex != -1, "Missing sections in result: %q", result)

	assert.True(t, systemIndex < factsIndex && factsIndex < contextIndex,
		"Incorrect section ordering. System: %d, Facts: %d, Context: %d", systemIndex, factsIndex, contextIndex)
}
