package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ethanbaker/assistant/internal/agents/overseer"
	"github.com/ethanbaker/assistant/internal/stores/memory"
	"github.com/ethanbaker/assistant/internal/stores/session"
	"github.com/ethanbaker/assistant/pkg/agent"
	"github.com/ethanbaker/assistant/pkg/utils"
	"github.com/go-sql-driver/mysql"
	"github.com/nlpodyssey/openai-agents-go/agents"
)

// Orchestrator is a wrapper for managing the agent's memory and session stores
type Orchestrator struct {
	memory   *memory.Store
	sessions session.Store
	overseer agent.CustomAgent
}

var orchestrator *Orchestrator

func main() {
	// Find env file
	envFile := ".env"
	if os.Getenv("ENV_FILE") != "" {
		envFile = os.Getenv("ENV_FILE")
	}

	// Load global config
	cfg := utils.NewConfigFromEnv(envFile)

	// Create MySQL config
	dbConfig := mysql.Config{
		User:      cfg.Get("MYSQL_USERNAME"),
		Passwd:    cfg.Get("MYSQL_ROOT_PASSWORD"),
		Net:       "tcp",
		Addr:      fmt.Sprintf("%s:%s", cfg.Get("MYSQL_HOST"), cfg.Get("MYSQL_PORT")),
		DBName:    cfg.Get("MYSQL_DATABASE"),
		ParseTime: true,
	}

	// Initialize database connections to create stores
	memoryStore, err := memory.NewStore(dbConfig.FormatDSN())
	if err != nil {
		log.Fatalf("[COMMANDLINE]: Failed to initialize memory store: %v", err)
	}

	sessionStore, err := session.NewMySqlStore(dbConfig.FormatDSN())
	if err != nil {
		log.Fatalf("[COMMANDLINE]: Failed to initialize session store: %v", err)
	}

	// Create overseer agent
	overseer, err := overseer.NewOverseerAgent(memoryStore, sessionStore, cfg)
	if err != nil {
		log.Fatalf("[COMMANDLINE]: Failed to initialize overseer agent: %v", err)
	}

	// Create the orchestrator with memory and session stores
	orchestrator = &Orchestrator{
		memory:   memoryStore,
		sessions: sessionStore,
		overseer: overseer,
	}

	// Start interactive session
	ctx := context.Background()
	if err := startInteractiveSession(ctx); err != nil {
		log.Fatalf("Failed to start interactive session: %v", err)
	}
}

// startInteractiveSession initializes the command line interface for the personal AI assistant
func startInteractiveSession(ctx context.Context) error {
	fmt.Println("Personal AI Assistant started. Type 'exit' to quit.")

	// Create a single session on startup for the entire conversation
	sess, err := orchestrator.sessions.CreateSession(ctx, "commandline-user")
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	fmt.Printf("Session created: %s\n", sess.SessionID(ctx))

	// Create scanner for reading user input
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\n> ")

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())

		if input == "exit" {
			break
		}

		if input == "" {
			continue
		}

		// Execute agent call with the persistent session
		response, err := executeAgentCall(ctx, sess, input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("Assistant: %s\n", response)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	return nil
}

func executeAgentCall(ctx context.Context, sess session.Session, input string) (string, error) {
	// Initialize OpenAI agents runner with the persistent session
	runner := agents.Runner{
		Config: agents.RunConfig{
			Session: sess,
		},
	}

	// Execute agent call
	response, err := runner.Run(ctx, orchestrator.overseer.Agent(), input)
	if err != nil {
		return "", fmt.Errorf("agent execution failed: %w", err)
	}

	// Convert response to string for display
	responseStr := fmt.Sprintf("%v", response.FinalOutput)
	return responseStr, nil
}
