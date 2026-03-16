package jobsubscription

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/ethanbaker/assistant/internal/domain"
	"gorm.io/gorm"
)

// InMemoryRepository stores subscriptions in memory.
type InMemoryRepository struct {
	mu     sync.RWMutex
	subs   map[int]*domain.JobSubscription
	nextID int
}

// NewInMemoryRepository creates a new in-memory subscription repository.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		subs:   make(map[int]*domain.JobSubscription),
		nextID: 1,
	}
}

func (r *InMemoryRepository) FindByClientAndJob(clientId, jobId int) (*domain.JobSubscription, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, sub := range r.subs {
		if sub.ClientId == clientId && sub.JobId == jobId {
			copy := *sub
			return &copy, nil
		}
	}
	return nil, nil
}

func (r *InMemoryRepository) FindActiveByJobId(jobId int) ([]*domain.JobSubscription, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*domain.JobSubscription, 0)
	for _, sub := range r.subs {
		if sub.JobId == jobId && sub.Active {
			copy := *sub
			result = append(result, &copy)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Priority < result[j].Priority
	})

	return result, nil
}

func (r *InMemoryRepository) Save(sub *domain.JobSubscription) error {
	if sub == nil {
		return fmt.Errorf("subscription is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	if sub.Model == nil {
		sub.Model = &gorm.Model{}
	}
	sub.ID = uint(r.nextID)
	sub.CreatedAt = now
	sub.UpdatedAt = now
	r.subs[r.nextID] = cloneSub(sub)
	r.nextID++
	return nil
}

func (r *InMemoryRepository) Update(sub *domain.JobSubscription) error {
	if sub == nil {
		return fmt.Errorf("subscription is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	id := int(sub.ID)
	if _, ok := r.subs[id]; !ok {
		return fmt.Errorf("subscription not found")
	}
	sub.UpdatedAt = time.Now().UTC()
	r.subs[id] = cloneSub(sub)
	return nil
}

func cloneSub(in *domain.JobSubscription) *domain.JobSubscription {
	copy := *in
	if in.Model != nil {
		m := *in.Model
		copy.Model = &m
	}
	return &copy
}
