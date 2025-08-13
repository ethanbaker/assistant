# Memory Package Tests

This directory contains comprehensive tests for the `pkg/memory` package, following the testing philosophy outlined in `docs/AGENT_TESTS.md` that emphasizes **functionality over coverage**.

## Testing Philosophy

The tests focus on:

1. **Interface behavior**: Testing that the `Store` and `KeyFact` types work as intended
2. **Real-world scenarios**: Testing how components work together in practical situations
3. **Edge cases and error handling**: Ensuring robust behavior under various conditions
4. **Database operations**: Verifying CRUD operations and search functionality

## Test Files

### `store_test.go`
Tests for the core Store functionality:

- **Database Operations**: Tests for CRUD operations (Create, Read, Update, Delete) on facts
- **Search Functionality**: Tests pattern matching and fact retrieval
- **Error Handling**: Tests behavior with invalid inputs, database errors, and context cancellation
- **Real-world Scenarios**: Tests user session management and data persistence patterns

Key test scenarios:
- Basic fact storage and retrieval
- Fact updates and overwrites
- Pattern-based searching with various query types
- Context handling and cancellation
- Error conditions (missing facts, database failures)
- Real-world user session simulation

### `types_test.go`
Tests for the `KeyFact` type functionality:

- **Type Creation**: Tests for `NewKeyFact` constructor
- **Field Validation**: Tests various key-value combinations
- **JSON Serialization**: Tests marshaling/unmarshaling behavior
- **Edge Cases**: Tests boundary conditions and special characters
- **Real-world Usage**: Tests practical usage patterns

Key test scenarios:
- Basic KeyFact creation with UUID generation
- Field validation with various data types
- JSON serialization/deserialization
- Unicode and special character handling
- Edge cases (empty values, long strings, special characters)
- Real-world data patterns (user preferences, configurations, etc.)

## Key Testing Patterns

### 1. Mock Database Approach
Since the actual database requires external dependencies, tests use a mock implementation:

```go
type mockDB struct {
    facts   map[string]*KeyFact
    closed  bool
    failOps bool // simulate database failures
}
```

This approach allows:
- Fast test execution without external dependencies
- Simulation of database failure scenarios
- Consistent test results across environments
- Easy testing of edge cases

### 2. Functionality-First Testing
Rather than testing every line of code, tests focus on:
- How the memory store is actually used in practice
- Real-world scenarios like user session management
- Integration between different store operations
- Error conditions that would affect users

### 3. Real-World Scenario Testing
The tests include comprehensive real-world scenarios:
- User session data management
- Preference storage and retrieval
- Configuration management
- Data cleanup and session management

### 4. Context-Aware Testing
Tests verify proper context handling:
- Context cancellation during operations
- Timeout behavior
- Proper error propagation

## Running the Tests

Run all memory tests:
```bash
go test ./pkg/memory/...
```

Run with verbose output:
```bash
go test -v ./pkg/memory/...
```

Run specific test functions:
```bash
go test -v ./pkg/memory/ -run TestStore_RealWorldScenario_UserSession
```

## Test Coverage Focus Areas

### Core Functionality
- ✅ KeyFact creation and validation
- ✅ Store CRUD operations (SetFact, GetFact, DeleteFact)
- ✅ Search functionality with pattern matching
- ✅ Query building and natural language processing

### Integration Scenarios
- ✅ User session data management
- ✅ Preference storage and updates
- ✅ Multi-fact operations and transactions
- ✅ Data consistency across operations

### Edge Cases
- ✅ Empty keys and values
- ✅ Special characters and Unicode
- ✅ Long strings and boundary conditions
- ✅ Database connection failures
- ✅ Context cancellation and timeouts

### Error Handling
- ✅ Invalid database URLs
- ✅ Missing fact retrieval
- ✅ Deletion of non-existent facts
- ✅ Database operation failures
- ✅ Network and connection issues

## Design Decisions

### Why Mock Database?
The tests use a mock database implementation instead of requiring a real database connection because:
- Tests run faster without external dependencies
- More reliable in CI/CD environments
- Easier to simulate failure conditions
- Focuses on testing business logic rather than database driver behavior

### Interface Testing Strategy
The tests verify the public interface behavior without testing internal implementation details:
- Tests focus on what the store does, not how it does it
- Makes tests resilient to refactoring
- Documents expected behavior clearly
- Catches bugs that affect actual usage

### Real-World Testing Approach
The tests emphasize realistic usage patterns:
- User session management scenarios
- Configuration and preference handling
- Data lifecycle operations (create, update, delete)
- Proper cleanup and resource management

This approach ensures the memory system works correctly in production scenarios and provides clear documentation of intended usage patterns.
