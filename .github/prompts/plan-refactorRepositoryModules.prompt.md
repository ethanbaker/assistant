# Plan: Refactor Repository Modules to Idiomatic Go Pattern

Refactoring three repository modules (memory, outreach, session) to follow a consistent, idiomatic Go pattern with interfaces and implementations. This standardizes the architecture, adds testing support via in-memory stores, ensures context propagation for cancellation/timeouts, and fixes broken import paths throughout the codebase.

**Key decisions:**

- Interfaces defined alongside implementations (not in pkg layer)
- All methods accept `context.Context` for cancellation/tracing
- Both MySQL and in-memory implementations for testing
- Fix broken `internal/stores/*` imports to `internal/repository/*`

## Steps

### 1. Refactor Memory Repository

**Location**: `internal/repository/memory/`

- Create `repository.go` with `Repository` interface defining:
  - `SetFact(ctx context.Context, key, value string) error`
  - `GetFact(ctx context.Context, key string) (*KeyFact, error)`
  - `SearchFacts(ctx context.Context, pattern string) ([]*KeyFact, error)`
  - `ListAllFacts(ctx context.Context) ([]*KeyFact, error)`
  - `DeleteFact(ctx context.Context, key string) error`
  - `Close() error`

- Rename `store.go` to `mysql.go`, rename `Store` struct to `MySQLRepository`
  - Update `NewStore()` to `NewMySQLRepository(databaseURL string) *MySQLRepository`
  - Make `BuildSearchQuery` private (rename to `buildSearchQuery`)

- Keep `keyfact.go` as-is (domain model)

- Create `memory.go` with `InMemoryRepository` implementing `Repository` interface
  - Use `map[string]*KeyFact` with `sync.RWMutex`
  - Include LIKE pattern matching for `SearchFacts`
  - Constructor: `NewInMemoryRepository() *InMemoryRepository`

### 2. Refactor Outreach Repository

**Location**: `internal/repository/outreach/`

- Create `repository.go` with `Repository` interface
  - Copy from `pkg/outreach/store.go` `StoreInterface`
  - Add `context.Context` to all method signatures:
    - `SaveImplementation(ctx context.Context, impl *Implementation) error`
    - `GetImplementation(ctx context.Context, clientID string) (*Implementation, error)`
    - `DisableImplementation(ctx context.Context, clientID string) error`
    - `ListImplementations(ctx context.Context) []*Implementation`
    - `Exists(ctx context.Context, clientID string) bool`
    - `AuthenticateImplementation(ctx context.Context, clientID, clientSecret string) (*Implementation, error)`
    - `Close() error`

- Rename `implementation.go` to `mysql.go`, rename `Store` to `MySQLRepository`
  - Update all methods to accept `ctx context.Context` as first parameter
  - Add `db.WithContext(ctx)` to all queries
  - **Fix bug in `ListImplementations`**: Move `.Where("active = ?", true)` before `.Find(&models)`
  - Update constructor to `NewMySQLRepository(databaseURL string) *MySQLRepository`

- Update `memory.go`: rename `InMemoryStore` to `InMemoryRepository`
  - Add context parameter to all methods (accept for interface compliance)
  - Update constructor to `NewInMemoryRepository() *InMemoryRepository`

- Keep `model.go` as-is (`ImplementationModel` for database)

### 3. Standardize Session Repository

**Location**: `internal/repository/session/`

- Create `repository.go`, move `Store` interface there and rename to `Repository`

- Rename in `store.go`:
  - `MySqlStore` → `MySQLRepository`
  - `InMemoryStore` → `InMemoryRepository`
  - `NewMySqlStore()` → `NewMySQLRepository()`
  - `NewInMemoryStore()` → `NewInMemoryRepository()`

- Update `session.go` references from `Store` to `Repository` in struct fields

- Keep `item.go`, `search.go` as-is

### 4. Update pkg Layer

**Remove duplicate interfaces**

- Delete `StoreInterface` from `pkg/outreach/store.go` (moved to repository layer)

- Update `pkg/outreach/manager.go`:
  - Imports: add `outreach_repo "internal/repository/outreach"`
  - Type references: `outreach.StoreInterface` → `outreach_repo.Repository`

- Update `pkg/outreach/types.go` if it references the old interface

### 5. Fix Broken Imports Across Codebase

**Change all `internal/stores/*` to `internal/repository/*`**

Files to update:

- `internal/agents/overseer/agent.go`
- `internal/agents/memory/agent.go`
- `internal/agents/task/agent.go`
- `internal/agents/communication/agent.go`
- `internal/agents/schedule/agent.go`
- `internal/agents/search/agent.go`
- `cmd/commandline/main.go`
- `internal/api/modules/agent/service-agent.go`
- `internal/api/modules/outreach/service-outreach.go`
- Any test files that import the old paths

Import changes:

- `internal/stores/memory` → `internal/repository/memory`
- `internal/stores/session` → `internal/repository/session`
- `internal/stores/outreach` → `internal/repository/outreach`

### 6. Update Constructor Calls and Type Declarations

**In all files from step 5:**

- Memory repository:
  - `memory.NewStore()` → `memory.NewMySQLRepository()`
  - `memory.Store` → `memory.Repository`

- Session repository:
  - `session.NewMySqlStore()` → `session.NewMySQLRepository()`
  - `session.Store` → `session.Repository`

- Outreach repository:
  - `outreach.NewStore()` → `outreach.NewMySQLRepository()`
  - For testing code using in-memory:
    - `session.NewInMemoryStore()` → `session.NewInMemoryRepository()`
    - `outreach.NewInMemoryStore()` → `outreach.NewInMemoryRepository()`

### 7. Verify Database Connections Still Work

- Ensure `internal/database/mysql.go` connection helper still works with renamed types
- Check `go.mod` dependencies are correct:
  - `gorm.io/driver/mysql`
  - `gorm.io/gorm`
  - `github.com/google/uuid`

## Verification

Run the following:

```bash
# Ensure all packages compile
go build ./...

# Run repository tests
go test ./internal/repository/...

# Run full test suite
go test ./...

# Start API server and verify database connections
./cmd/api/main.go

# Test CLI tool with repositories
./cmd/commandline/main.go
```

Manual checks:

- Check logs for successful MySQL migrations on startup
- Manually test one CRUD operation per repository via API or CLI

## Key Design Decisions

1. **Interface Location**: Interfaces in repository packages (not pkg/) for co-location with implementations

2. **Context Pattern**: Context-first parameter pattern for cancellation and distributed tracing support

3. **Naming Convention**: `MySQLRepository` / `InMemoryRepository` for consistency across all modules

4. **Bug Fix**: Fixed outreach `ListImplementations` bug (WHERE clause ordering)

## Reference Implementation

Following this pattern from the external example:

```go
type Repository interface {
    FindPublished(ctx context.Context) ([]domain.Project, error)
    FindByID(ctx context.Context, id int64) (*domain.Project, error)
    ExistsBySlug(ctx context.Context, s string) (bool, error)
    Create(ctx context.Context, p *domain.Project) (*domain.Project, error)
    Update(ctx context.Context, p *domain.Project) (*domain.Project, error)
    Delete(ctx context.Context, id int64) error
}

type MySQLRepository struct {
    db *gorm.DB
}

func NewMySQLRepository(db *gorm.DB) *MySQLRepository {
    return &MySQLRepository{db: db}
}
```
