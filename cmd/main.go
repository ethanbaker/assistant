package main

import (
	"context"
	"fmt"
	"log"

	overseeragent "github.com/ethanbaker/assistant/internal/overseer-agent"
	"github.com/ethanbaker/assistant/pkg/agent"
	"github.com/ethanbaker/assistant/pkg/memory"
	"github.com/ethanbaker/assistant/pkg/session"
	"github.com/ethanbaker/assistant/pkg/utils"
	"github.com/nlpodyssey/openai-agents-go/agents"
)

type Orchestrator struct {
	memoryStore   *memory.Store
	sessionStore  *session.Store
	overseerAgent agent.CustomAgent
}

func main() {
	// Load global configuration
	cfg := utils.NewConfigFromEnv(".env")

	// Initialize database connections
	memoryStore, err := memory.NewStore(cfg.Get("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Failed to initialize memory store: %v", err)
	}

	sessionStore, err := session.NewStore(cfg.Get("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Failed to initialize session store: %v", err)
	}

	// Initialize orchestrator
	orchestrator := &Orchestrator{
		memoryStore:  memoryStore,
		sessionStore: sessionStore,
	}

	// Initialize overseer agent
	orchestrator.initializeOverseerAgent()

	// Start interactive session
	ctx := context.Background()
	if err := orchestrator.startInteractiveSession(ctx); err != nil {
		log.Fatalf("Failed to start interactive session: %v", err)
	}
}

func (o *Orchestrator) initializeOverseerAgent() {
	// Create the overseer agent with all specialized agents as handoffs
	o.overseerAgent = overseeragent.NewOverseerAgent(o.memoryStore, o.sessionStore)
}

func (o *Orchestrator) startInteractiveSession(ctx context.Context) error {
	fmt.Println("Personal AI Assistant started. Type 'exit' to quit.")

	// Create new session
	sess, err := o.sessionStore.CreateSession(ctx, "user1")
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	for {
		fmt.Print("\n> ")
		var input string
		if _, err := fmt.Scanln(&input); err != nil {
			continue
		}

		if input == "exit" {
			break
		}

		// Route through overseer agent
		response, err := o.executeAgentCall(ctx, sess, o.overseerAgent, input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("Assistant: %s\n", response)
	}

	return nil
}

func (o *Orchestrator) executeAgentCall(ctx context.Context, sess *session.Session, targetAgent agent.CustomAgent, input string) (string, error) {
	// Save user message
	userMsg := &session.Message{
		SessionID: sess.ID,
		Role:      "user",
		Content:   input,
	}
	if err := o.sessionStore.SaveMessage(ctx, userMsg); err != nil {
		return "", fmt.Errorf("failed to save user message: %w", err)
	}

	// Execute agent call
	response, err := agents.Run(ctx, targetAgent.Agent(), input)
	if err != nil {
		return "", fmt.Errorf("agent execution failed: %w", err)
	}

	// Convert response to string for message storage
	responseStr := fmt.Sprintf("%v", response)

	// Save assistant response
	assistantMsg := &session.Message{
		SessionID: sess.ID,
		Role:      "assistant",
		Content:   responseStr,
	}
	if err := o.sessionStore.SaveMessage(ctx, assistantMsg); err != nil {
		return "", fmt.Errorf("failed to save assistant message: %w", err)
	}

	return responseStr, nil
}
