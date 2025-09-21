package outreach

import (
	"fmt"
	"sync"

	"github.com/ethanbaker/assistant/pkg/outreach"
)

// InMemoryStore provides an in-memory implementation of StoreInterface for testing
type InMemoryStore struct {
	implementations map[string]*outreach.Implementation
	mutex           sync.RWMutex
}

// NewInMemoryStore creates a new in-memory outreach store
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		implementations: make(map[string]*outreach.Implementation),
		mutex:           sync.RWMutex{},
	}
}

// SaveImplementation stores an implementation by client ID
func (s *InMemoryStore) SaveImplementation(impl *outreach.Implementation) error {
	if impl.ClientID == "" {
		return fmt.Errorf("client_id cannot be empty")
	}
	if impl.CallbackURL == "" {
		return fmt.Errorf("callback_url cannot be empty")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Create a copy to avoid shared references
	implCopy := &outreach.Implementation{
		ClientID:     impl.ClientID,
		CallbackURL:  impl.CallbackURL,
		ClientSecret: impl.ClientSecret,
	}

	s.implementations[impl.ClientID] = implCopy
	return nil
}

// GetImplementation retrieves an implementation by client ID
func (s *InMemoryStore) GetImplementation(clientID string) (*outreach.Implementation, error) {
	if clientID == "" {
		return nil, fmt.Errorf("client_id cannot be empty")
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	impl, exists := s.implementations[clientID]
	if !exists {
		return nil, fmt.Errorf("implementation with client_id '%s' not found", clientID)
	}

	// Return a copy to avoid external mutations
	return &outreach.Implementation{
		ClientID:     impl.ClientID,
		CallbackURL:  impl.CallbackURL,
		ClientSecret: impl.ClientSecret,
	}, nil
}

// DisableImplementation removes an implementation by client ID
func (s *InMemoryStore) DisableImplementation(clientID string) error {
	if clientID == "" {
		return fmt.Errorf("client_id cannot be empty")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.implementations[clientID]; !exists {
		return fmt.Errorf("implementation with client_id '%s' not found", clientID)
	}

	s.implementations[clientID].Active = false
	return nil
}

// ListImplementations returns all registered implementations
func (s *InMemoryStore) ListImplementations() []*outreach.Implementation {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	implementations := make([]*outreach.Implementation, 0, len(s.implementations))
	for _, impl := range s.implementations {
		// Create copies to avoid external mutations
		implementations = append(implementations, &outreach.Implementation{
			ClientID:     impl.ClientID,
			CallbackURL:  impl.CallbackURL,
			ClientSecret: impl.ClientSecret,
		})
	}

	return implementations
}

// Exists checks if an implementation with the given client ID exists
func (s *InMemoryStore) Exists(clientID string) bool {
	if clientID == "" {
		return false
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	_, exists := s.implementations[clientID]
	return exists
}

// AuthenticateImplementation validates client ID and secret combination
func (s *InMemoryStore) AuthenticateImplementation(clientID, clientSecret string) (*outreach.Implementation, error) {
	if clientID == "" {
		return nil, fmt.Errorf("client_id cannot be empty")
	}
	if clientSecret == "" {
		return nil, fmt.Errorf("client_secret cannot be empty")
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	impl, exists := s.implementations[clientID]
	if !exists {
		return nil, fmt.Errorf("invalid client credentials")
	}

	if impl.ClientSecret != clientSecret {
		return nil, fmt.Errorf("invalid client credentials")
	}

	// Return a copy to avoid shared references
	return &outreach.Implementation{
		ClientID:     impl.ClientID,
		CallbackURL:  impl.CallbackURL,
		ClientSecret: impl.ClientSecret,
	}, nil
}
