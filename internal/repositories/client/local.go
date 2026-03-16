package client

import (
	"fmt"
	"sync"
	"time"

	"github.com/ethanbaker/assistant/internal/domain"
	"gorm.io/gorm"
)

// InMemoryRepository stores clients in memory.
type InMemoryRepository struct {
	mu      sync.RWMutex
	clients map[int]*domain.Client
	nextID  int
}

// NewInMemoryRepository creates a new in-memory client repository.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		clients: make(map[int]*domain.Client),
		nextID:  1,
	}
}

func (r *InMemoryRepository) FindById(id int) (*domain.Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	client, ok := r.clients[id]
	if !ok {
		return nil, nil
	}
	copy := *client
	return &copy, nil
}

func (r *InMemoryRepository) FindByApiKey(apiKey string) (*domain.Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, client := range r.clients {
		if client.ApiKey == apiKey {
			copy := *client
			return &copy, nil
		}
	}
	return nil, nil
}

func (r *InMemoryRepository) FindByName(name string) (*domain.Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, client := range r.clients {
		if client.Name == name {
			copy := *client
			return &copy, nil
		}
	}
	return nil, nil
}

func (r *InMemoryRepository) Save(client *domain.Client) error {
	if client == nil {
		return fmt.Errorf("client is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	if client.Model == nil {
		client.Model = &gorm.Model{}
	}
	client.ID = uint(r.nextID)
	client.CreatedAt = now
	client.UpdatedAt = now
	r.clients[r.nextID] = cloneClient(client)
	r.nextID++
	return nil
}

func cloneClient(in *domain.Client) *domain.Client {
	copy := *in
	if in.Model != nil {
		m := *in.Model
		copy.Model = &m
	}
	if in.Subscriptions != nil {
		copy.Subscriptions = append([]domain.JobSubscription(nil), in.Subscriptions...)
	}
	return &copy
}
