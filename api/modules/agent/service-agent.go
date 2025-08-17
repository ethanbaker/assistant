package agent

import (
	"context"
	"fmt"
	"log"

	overseeragent "github.com/ethanbaker/assistant/internal/overseer-agent"
	"github.com/ethanbaker/assistant/pkg/agent"
	"github.com/ethanbaker/assistant/pkg/memory"
	"github.com/ethanbaker/assistant/pkg/session"
	"github.com/ethanbaker/assistant/pkg/utils"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/nlpodyssey/openai-agents-go/agents"
)

// Orchestrator is a wrapper for managing the agent's memory and session stores
type Orchestrator struct {
	memory   *memory.Store
	sessions *session.Store
	overseer agent.CustomAgent
}

var orchestrator *Orchestrator

// Create an assistant for the api to run off of
func Init(cfg *utils.Config) {
	// Create MySQL config
	dbConfig := mysql.Config{
		User:      cfg.Get("MYSQL_USER"),
		Passwd:    cfg.Get("MYSQL_ROOT_PASSWORD"),
		Net:       "tcp",
		Addr:      fmt.Sprintf("%s:%s", cfg.Get("MYSQL_HOST"), cfg.Get("MYSQL_PORT")),
		DBName:    cfg.Get("MYSQL_DATABASE"),
		ParseTime: true,
	}

	// Initialize database connections to create stores
	memoryStore, err := memory.NewStore(dbConfig.FormatDSN())
	if err != nil {
		log.Fatalf("[AGENT]: Failed to initialize memory store: %v", err)
	}

	sessionStore, err := session.NewStore(dbConfig.FormatDSN())
	if err != nil {
		log.Fatalf("[AGENT]: Failed to initialize session store: %v", err)
	}

	// Create overseer agent
	overseer, err := overseeragent.NewOverseerAgent(memoryStore, sessionStore, cfg)
	if err != nil {
		log.Fatalf("[AGENT]: Failed to initialize overseer agent: %v", err)
	}

	// Create the orchestrator with memory and session stores
	orchestrator = &Orchestrator{
		memory:   memoryStore,
		sessions: sessionStore,
		overseer: overseer,
	}
}

// Return the orchestrator instance
func GetOrchestrator() *Orchestrator {
	if orchestrator == nil {
		log.Fatal("[AGENT]: Orchestrator is not initialized")
	}
	return orchestrator
}

// Create a new session
func (o *Orchestrator) NewSession(ctx context.Context, userID string) (*session.Session, error) {
	return o.sessions.CreateSession(ctx, userID)
}

// Find an existing session by UUID
func (o *Orchestrator) FindSession(ctx context.Context, sessionID string) (*session.Session, error) {
	// Validate the session ID format
	guid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID format: %v", err)
	}

	return o.sessions.GetSessionWithItems(ctx, guid)
}

// Add a message to an existing session
func (o *Orchestrator) AddMessage(ctx context.Context, sessionID string, req PostMessageRequest) (*agents.RunResult, error) {
	// Parse the session ID
	guid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID format: %v", err)
	}

	// Find the session
	sess, err := o.sessions.GetSession(ctx, guid)
	if err != nil {
		return nil, err
	}

	// Add data to the context
	if req.Data != nil {
		ctx = context.WithValue(ctx, "data", req.Data)
	}

	// Initialize OpenAI agents runner
	runner := agents.Runner{
		Config: agents.RunConfig{
			Session: sess,
		},
	}

	// Execute agent call
	resp, err := runner.Run(ctx, o.overseer.Agent(), req.Content)
	if err != nil {
		return nil, fmt.Errorf("agent execution failed: %w", err)
	}

	// Return response
	return resp, nil
}

// Remove an existing session and return it
func (o *Orchestrator) RemoveSession(ctx context.Context, sessionID string) (*session.Session, error) {
	// Parse session ID
	guid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID format: %v", err)
	}

	// Get the session to return it
	sess, err := o.sessions.GetSessionWithItems(ctx, guid)
	if err != nil {
		return nil, err
	}

	// Delete the session from the database
	if err := o.sessions.DeleteSession(ctx, guid); err != nil {
		return nil, err
	}

	return sess, nil
}
