package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ethanbaker/assistant/internal/domain"
	"github.com/ethanbaker/assistant/internal/prompts"
	jobexecutionrepo "github.com/ethanbaker/assistant/internal/repositories/jobexecution"
	"github.com/ethanbaker/assistant/internal/repositories/session"
	"github.com/ethanbaker/assistant/pkg/config"
	"github.com/ethanbaker/assistant/pkg/logger"
	"github.com/ethanbaker/assistant/pkg/sdk"
	"github.com/google/uuid"
	"github.com/nlpodyssey/openai-agents-go/agents"
	"github.com/nlpodyssey/openai-agents-go/memory"
)

// ServiceConfig
type ServiceConfig struct {
	SessionRepository   session.Repository
	ExecutionRepository jobexecutionrepo.Repository
	EntryAgent          domain.CustomAgent
}

// Service defines dependencies for the agent service
type Service struct {
	ServiceConfig
}

var (
	ErrExecutionNotFound  = errors.New("job execution not found")
	ErrExecutionForbidden = errors.New("job execution does not belong to client")
)

// NewService creates a new agent service instance
func NewService(cfg ServiceConfig) *Service {
	return &Service{
		ServiceConfig: cfg,
	}
}

// CreateSession creates a new session
func (s *Service) CreateSession(ctx context.Context, userID string) (memory.Session, error) {
	logger.Debugf("CreateSession called for userID: %s", userID)
	sess, err := s.SessionRepository.CreateSession(ctx, userID)
	if err != nil {
		logger.Debugf("CreateSession failed for userID %s: %v", userID, err)
		return nil, err
	}
	logger.Debugf("CreateSession succeeded for userID: %s", userID)
	return sess, nil
}

// GetSession finds an existing session by UUID
func (s *Service) GetSession(ctx context.Context, sessionID string) (memory.Session, error) {
	logger.Debugf("GetSession called for sessionID: %s", sessionID)
	// Validate the session ID format
	guid, err := uuid.Parse(sessionID)
	if err != nil {
		logger.Debugf("GetSession failed: invalid session ID format: %v", err)
		return nil, fmt.Errorf("invalid session ID format: %v", err)
	}

	logger.Debugf("GetSession retrieving session with items for GUID: %v", guid)
	sess, err := s.SessionRepository.GetSessionWithItems(ctx, guid)
	if err != nil {
		logger.Debugf("GetSession failed to retrieve session: %v", err)
		return nil, err
	}
	logger.Debugf("GetSession succeeded for sessionID: %s", sessionID)
	return sess, nil
}

// AddMessage adds a message to an existing session
func (s *Service) AddMessage(ctx context.Context, sessionID string, req sdk.PostMessageRequest) (*agents.RunResult, error) {
	logger.Debugf("AddMessage called for sessionID: %s with content length: %d", sessionID, len(req.Content))
	// Parse the session ID
	guid, err := uuid.Parse(sessionID)
	if err != nil {
		logger.Debugf("AddMessage failed: invalid session ID format: %v", err)
		return nil, fmt.Errorf("invalid session ID format: %v", err)
	}

	// Find the session
	logger.Debugf("AddMessage retrieving session for GUID: %v", guid)
	sess, err := s.SessionRepository.GetSession(ctx, guid)
	if err != nil {
		logger.Debugf("AddMessage failed to retrieve session: %v", err)
		return nil, err
	}

	// Add data to the context
	if req.Data != nil {
		logger.Debugf("AddMessage adding data to context")
		ctx = context.WithValue(ctx, "data", req.Data)
	}

	contextLimit, _ := config.Getenv("CONTEXT_LIMIT")
	limit := 10 // default
	if contextLimit != "" {
		fmt.Sscanf(contextLimit, "%d", &limit)
	}
	logger.Debugf("AddMessage using context limit: %d", limit)

	// Add relevant context to session
	if err := prompts.InjectCurrentTimeContext(ctx, sess, time.Now()); err != nil {
		return nil, fmt.Errorf("Failed to inject current time: %v", err)
	}

	// Initialize OpenAI agents runner
	logger.Debugf("AddMessage initializing runner with limit: %d", limit)
	runner := agents.Runner{
		Config: agents.RunConfig{
			Session:     sess,
			LimitMemory: limit,
		},
	}

	// Execute agent call
	logger.Debugf("AddMessage executing agent run")
	resp, err := runner.Run(ctx, s.EntryAgent.Agent(), req.Content)
	if err != nil {
		logger.Debugf("AddMessage agent execution failed: %v", err)
		return nil, fmt.Errorf("agent execution failed: %w", err)
	}

	// Return response
	logger.Debugf("AddMessage succeeded for sessionID: %s", sessionID)
	return resp, nil
}

// DeleteSession removes an existing session and returns it
func (s *Service) DeleteSession(ctx context.Context, sessionID string) (memory.Session, error) {
	logger.Debugf("DeleteSession called for sessionID: %s", sessionID)
	// Parse session ID
	guid, err := uuid.Parse(sessionID)
	if err != nil {
		logger.Debugf("DeleteSession failed: invalid session ID format: %v", err)
		return nil, fmt.Errorf("invalid session ID format: %v", err)
	}

	// Get the session to return it
	logger.Debugf("DeleteSession retrieving session for GUID: %v", guid)
	sess, err := s.SessionRepository.GetSessionWithItems(ctx, guid)
	if err != nil {
		logger.Debugf("DeleteSession failed to retrieve session: %v", err)
		return nil, err
	}

	// Delete the session from the database
	logger.Debugf("DeleteSession deleting session from database")
	if err := s.SessionRepository.DeleteSession(ctx, guid); err != nil {
		logger.Debugf("DeleteSession failed to delete session: %v", err)
		return nil, err
	}

	logger.Debugf("DeleteSession succeeded for sessionID: %s", sessionID)
	return sess, nil
}

// GetSessionItems gets the latest 'n' items from the session
func (s *Service) GetSessionItems(ctx context.Context, sessionID string, limit int) ([]*domain.Item, error) {
	logger.Debugf("GetSessionItems called for sessionID: %s with limit: %d", sessionID, limit)
	// Parse session ID
	guid, err := uuid.Parse(sessionID)
	if err != nil {
		logger.Debugf("GetSessionItems failed: invalid session ID format: %v", err)
		return nil, fmt.Errorf("invalid session ID format: %v", err)
	}

	// Get items from repository
	logger.Debugf("GetSessionItems retrieving items for GUID: %v", guid)
	allItems, err := s.SessionRepository.GetSessionItems(ctx, guid)
	if err != nil {
		logger.Debugf("GetSessionItems failed to retrieve items: %v", err)
		return nil, err
	}

	logger.Debugf("GetSessionItems retrieved %d total items", len(allItems))

	// Return the latest N items
	if limit > 0 && len(allItems) > limit {
		logger.Debugf("GetSessionItems returning latest %d of %d items", limit, len(allItems))
		return allItems[len(allItems)-limit:], nil
	}

	logger.Debugf("GetSessionItems succeeded, returning %d items", len(allItems))
	return allItems, nil
}

// GetItemCount gets the number of items in a session
func (s *Service) GetItemCount(ctx context.Context, sessionID string) (int, error) {
	logger.Debugf("GetItemCount called for sessionID: %s", sessionID)
	// Parse session ID
	guid, err := uuid.Parse(sessionID)
	if err != nil {
		logger.Debugf("GetItemCount failed: invalid session ID format: %v", err)
		return 0, fmt.Errorf("invalid session ID format: %v", err)
	}

	// Get items from repository
	logger.Debugf("GetItemCount retrieving items for GUID: %v", guid)
	items, err := s.SessionRepository.GetSessionItems(ctx, guid)
	if err != nil {
		logger.Debugf("GetItemCount failed to retrieve items: %v", err)
		return 0, err
	}

	count := len(items)
	logger.Debugf("GetItemCount succeeded for sessionID %s: %d items", sessionID, count)
	return count, nil
}

// AttachJobExecutionsContext injects job execution payloads into a session as context items.
func (s *Service) AttachJobExecutionsContext(ctx context.Context, sessionID string, clientID int, executionIDs []int) error {
	if s.ExecutionRepository == nil {
		return fmt.Errorf("execution repository not configured")
	}

	if len(executionIDs) == 0 {
		return fmt.Errorf("job_execution_ids is required")
	}

	guid, err := uuid.Parse(sessionID)
	if err != nil {
		return fmt.Errorf("invalid session ID format: %v", err)
	}

	sess, err := s.SessionRepository.GetSession(ctx, guid)
	if err != nil {
		return err
	}

	for _, executionID := range executionIDs {
		execution, findErr := s.ExecutionRepository.FindById(executionID)
		if findErr != nil {
			return fmt.Errorf("failed to find job execution %d: %w", executionID, findErr)
		}
		if execution == nil {
			return ErrExecutionNotFound
		}
		if execution.ClientId != clientID {
			return ErrExecutionForbidden
		}

		serialized, marshalErr := json.Marshal(execution)
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal execution %d: %w", executionID, marshalErr)
		}

		content := fmt.Sprintf("Outreach job execution context (id=%d): %s", executionID, string(serialized))
		if injectErr := prompts.InjectContextItem(ctx, sess, content); injectErr != nil {
			return fmt.Errorf("failed to inject execution %d into session: %w", executionID, injectErr)
		}
	}

	return nil
}
