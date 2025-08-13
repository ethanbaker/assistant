package agent

import (
	"context"
	"testing"

	"github.com/ethanbaker/assistant/pkg/utils"
	"github.com/nlpodyssey/openai-agents-go/agents"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAgent implements CustomAgent for testing
type mockAgent struct {
	id     string
	config *utils.Config
	agent  *agents.Agent
	dryRun bool
}

func (m *mockAgent) Agent() *agents.Agent {
	return m.agent
}

func (m *mockAgent) ID() string {
	return m.id
}

func (m *mockAgent) Config() *utils.Config {
	return m.config
}

func (m *mockAgent) ShouldDryRun(ctx context.Context) bool {
	return m.dryRun
}

// mockDynamicPromptAgent implements DynamicPromptAgent for testing
type mockDynamicPromptAgent struct {
	mockAgent
	promptTemplate string
}

func (m *mockDynamicPromptAgent) DynamicPrompt(session any) string {
	return m.promptTemplate + " with session data"
}

// Test CustomAgent interface implementation
func TestCustomAgent_Interface(t *testing.T) {
	tests := []struct {
		name    string
		agent   CustomAgent
		wantID  string
		wantDry bool
	}{
		{
			name: "basic custom agent implementation",
			agent: &mockAgent{
				id:     "test-agent",
				config: utils.NewConfig(map[string]string{"key": "value"}),
				dryRun: false,
			},
			wantID:  "test-agent",
			wantDry: false,
		},
		{
			name: "agent with dry run enabled",
			agent: &mockAgent{
				id:     "dry-run-agent",
				config: utils.NewConfig(map[string]string{"key": "value"}),
				dryRun: true,
			},
			wantID:  "dry-run-agent",
			wantDry: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test ID functionality
			assert.Equal(t, tt.wantID, tt.agent.ID())

			// Test Config functionality
			config := tt.agent.Config()
			require.NotNil(t, config)

			// Test ShouldDryRun functionality
			ctx := context.Background()
			assert.Equal(t, tt.wantDry, tt.agent.ShouldDryRun(ctx))

			// Test Agent method returns non-nil (even if it's nil in mock)
			_ = tt.agent.Agent()
		})
	}
}

// Test DynamicPromptAgent interface implementation
func TestDynamicPromptAgent_Interface(t *testing.T) {
	agent := &mockDynamicPromptAgent{
		mockAgent: mockAgent{
			id:     "dynamic-agent",
			config: utils.NewConfig(map[string]string{}),
			dryRun: false,
		},
		promptTemplate: "Base prompt",
	}

	// Test that it implements CustomAgent
	var customAgent CustomAgent = agent
	assert.Equal(t, "dynamic-agent", customAgent.ID())

	// Test dynamic prompt functionality
	sessionData := map[string]string{"user": "test"}
	prompt := agent.DynamicPrompt(sessionData)
	expected := "Base prompt with session data"
	assert.Equal(t, expected, prompt)
}

// Test that DynamicPromptAgent can be used as CustomAgent
func TestAgent_InterfaceComposition(t *testing.T) {
	// Test that DynamicPromptAgent properly embeds CustomAgent
	dynamicAgent := &mockDynamicPromptAgent{
		mockAgent: mockAgent{
			id:     "composed-agent",
			config: utils.NewConfig(map[string]string{"test": "value"}),
			dryRun: true,
		},
		promptTemplate: "Dynamic template",
	}

	// Should be able to use as CustomAgent
	var agent CustomAgent = dynamicAgent

	assert.Equal(t, "composed-agent", agent.ID())

	ctx := context.Background()
	assert.True(t, agent.ShouldDryRun(ctx))

	config := agent.Config()
	assert.Equal(t, "value", config.Get("test"))
}

// Test that agents handle context properly
func TestAgent_ContextBehavior(t *testing.T) {
	agent := &mockAgent{
		id:     "context-agent",
		config: utils.NewConfig(map[string]string{}),
		dryRun: false,
	}

	tests := []struct {
		name    string
		ctx     context.Context
		wantErr bool
	}{
		{
			name:    "with background context",
			ctx:     context.Background(),
			wantErr: false,
		},
		{
			name:    "with cancelled context",
			ctx:     func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }(),
			wantErr: false, // ShouldDryRun doesn't check context cancellation in our mock
		},
		{
			name:    "with context values",
			ctx:     context.WithValue(context.Background(), struct{ key string }{"test"}, "value"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This demonstrates that agents should handle context properly
			// In real implementations, agents might use context for timeouts, cancellation, etc.
			result := agent.ShouldDryRun(tt.ctx)
			// The result should be consistent regardless of context in our mock
			assert.Equal(t, agent.dryRun, result, "ShouldDryRun() inconsistent with context %v", tt.name)
		})
	}
}
