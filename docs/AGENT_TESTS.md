# Agent Package Tests

This directory contains comprehensive tests for the `pkg/agent` package, focusing on functionality and real-world usage rather than just code coverage.

## Testing Philosophy

The tests emphasize **functionality over coverage**, meaning they focus on:

1. **Interface behavior**: Testing that the `CustomAgent` and `DynamicPromptAgent` interfaces work as intended
2. **Real-world scenarios**: Testing how components work together in practical situations
3. **Edge cases and error handling**: Ensuring robust behavior under various conditions
4. **Integration testing**: Verifying that different parts of the system work together correctly

## Test Files

### `agent_test.go`
Tests for the core agent interfaces:

- **Interface Implementation**: Verifies that the `CustomAgent` interface works correctly
- **Dynamic Prompt Agents**: Tests the `DynamicPromptAgent` interface and its composition with `CustomAgent`
- **Context Handling**: Ensures agents properly handle Go contexts for cancellation, timeouts, etc.
- **Interface Composition**: Tests that embedded interfaces work correctly

Key test scenarios:
- Basic agent functionality (ID, config, dry-run behavior)
- Dynamic prompt generation with session data
- Context-aware behavior
- Interface composition and inheritance

### `prompt_test.go`
Tests for the `PromptBuilder` functionality:

- **Builder Pattern**: Tests the fluent interface and method chaining
- **Dynamic Prompt Construction**: Verifies that prompts are built correctly with facts and context
- **Section Ordering**: Ensures proper order of system prompt, facts, and context
- **Edge Cases**: Handles special characters, empty values, and complex scenarios

Key test scenarios:
- Basic prompt building with system prompts
- Adding contextual information and facts
- Method chaining for fluent interface
- Complex prompts with all sections
- Edge cases (special characters, empty values, overwrites)
- Section ordering and formatting

### `config_test.go`
Tests for the `LoadAgentConfig` functionality:

- **Configuration Loading**: Tests loading of agent-specific and global configurations
- **Precedence Rules**: Verifies that agent-specific configs override global ones
- **Environment Integration**: Tests interaction with environment variables
- **Isolation**: Ensures different agents don't interfere with each other's configurations

Key test scenarios:
- Agent-specific configuration precedence
- Fallback to global configuration
- Environment variable integration
- Configuration isolation between agents
- Real-world integration scenarios
- Edge cases (missing files, empty names)

## Key Testing Patterns

### 1. Environment Isolation
Tests ensure that environment variables and configuration files from one test don't affect others:

```go
// Store original environment variables
originalEnvVars := make(map[string]string)
for key := range testKeys {
    if val := os.Getenv(key); val != "" {
        originalEnvVars[key] = val
    }
    os.Unsetenv(key) // Clear for test isolation
}

defer func() {
    // Restore original environment
    for key, val := range originalEnvVars {
        os.Setenv(key, val)
    }
}()
```

### 2. Functionality-First Testing
Rather than testing every line of code, tests focus on:
- How the components are actually used
- Real-world scenarios and use cases
- Integration between different parts
- Error conditions and edge cases

### 3. Test Helper Functions
Common functionality is extracted into helper functions:
- `createEnvFile()`: Creates .env files with specific content
- Mock implementations for testing interfaces
- Environment cleanup and restoration

## Running the Tests

Run all agent tests:
```bash
go test ./pkg/agent/...
```

Run with verbose output:
```bash
go test -v ./pkg/agent/...
```

Run specific test functions:
```bash
go test -v ./pkg/agent/ -run TestPromptBuilder_ChainedMethods
```

## Test Coverage Focus Areas

### Core Functionality
- ✅ `PromptBuilder` creation and building
- ✅ `LoadAgentConfig` with precedence rules
- ✅ Interface implementations and composition
- ✅ Environment variable handling

### Integration Scenarios
- ✅ Agent-specific vs global configuration
- ✅ Dynamic prompt generation with session data
- ✅ Context handling in agent operations
- ✅ Multi-agent configuration isolation

### Edge Cases
- ✅ Empty configurations and prompts
- ✅ Special characters in configuration values
- ✅ Missing configuration files
- ✅ Environment variable precedence
- ✅ Configuration overwrites and updates

## Design Decisions

### Why Focus on Functionality?
These tests prioritize testing **what the code does** rather than **how it does it**. This approach:
- Makes tests more resilient to implementation changes
- Focuses on user-facing behavior
- Catches real bugs that affect functionality
- Documents expected behavior clearly

### Interface Testing Strategy
The tests use mock implementations to verify interface contracts without depending on external systems. This ensures:
- Fast test execution
- Reliable test results
- Clear interface documentation
- Easy testing of edge cases

### Configuration Testing Approach
Configuration tests emphasize real-world scenarios:
- Multiple agents with different configurations
- Precedence rules between global and agent-specific settings
- Environment variable integration
- Proper isolation between test runs

This approach ensures the configuration system works correctly in production scenarios.
