# Overseer Agent Implementation

## Overview

The Personal AI Assistant now uses an **Overseer Agent** architecture that leverages the `WithHandoffs` method from the `openai-agents-go` library. This provides a more intelligent and automated way to route user requests to the appropriate specialized agents.

## Architecture Changes

### Before (Manual Routing)
- The main orchestrator used simple keyword matching to route requests
- Direct instantiation and management of all specialized agents
- Manual intent detection logic

### After (Overseer Agent with Handoffs)
- Single overseer agent that understands user intent through LLM reasoning
- Automatic handoffs to specialized agents using the `WithHandoffs` pattern
- More intelligent routing that can handle complex or ambiguous requests

## Implementation Details

### Overseer Agent (`internal/overseer-agent/agent.go`)

The overseer agent is implemented with:

1. **HandoffFromAgent Creation**: Each specialized agent is converted to a handoff using `agents.HandoffFromAgent()`
2. **Descriptive Tool Names**: Each handoff has a clear tool name and description
3. **Comprehensive Instructions**: The overseer knows when and how to delegate to each agent

```go
// Example handoff creation
memoryHandoff := agents.HandoffFromAgent(agents.HandoffFromAgentParams{
    Agent:                   memoryAgent.Agent(),
    ToolNameOverride:        "handoff_to_memory_agent",
    ToolDescriptionOverride: "Hand off to the Memory Agent for storing facts, recalling information, or searching past conversations",
})
```

### Available Handoffs

The overseer agent can hand off to:

1. **Memory Agent** - For facts, memories, and conversation history
2. **GitHub Agent** - For repository management and code-related tasks  
3. **Notion Agent** - For note-taking and document management
4. **Email Agent** - For email operations and inbox management

### Main Application Changes (`cmd/main.go`)

- Simplified orchestrator structure with just the overseer agent
- Removed manual routing logic
- All requests go directly to the overseer agent
- The overseer agent intelligently determines which specialized agent should handle each request

## Benefits

1. **Smarter Routing**: Uses LLM reasoning instead of keyword matching
2. **Better Context Handling**: Can understand complex requests that span multiple domains
3. **Easier Maintenance**: No need to update routing rules when adding new agents
4. **More Natural Conversations**: Can handle follow-up questions and context switches

## Usage

The usage remains the same for end users:

```bash
# Start the assistant
make run

# Examples that will be automatically routed:
> "Remember my birthday is March 15th"        # → Memory Agent
> "What are my GitHub repositories?"          # → GitHub Agent  
> "Create a note about today's meeting"       # → Notion Agent
> "Check my recent emails"                    # → Email Agent
> "What was that GitHub issue I mentioned yesterday?" # → Memory Agent (searches history) → GitHub Agent (if needed)
```

## Extension

To add new agents:

1. Create the agent in `internal/your-agent/`
2. Add it as a handoff in `NewOverseerAgent()`
3. Update the overseer's instructions to describe when to use the new agent
4. No changes needed to main.go routing logic

The overseer agent will automatically learn to use new handoffs based on their descriptions.
