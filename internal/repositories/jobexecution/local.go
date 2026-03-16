package jobexecution

import (
	"fmt"
	"sync"
	"time"

	"github.com/ethanbaker/assistant/internal/domain"
	"gorm.io/gorm"
)

// InMemoryRepository stores job executions in memory.
type InMemoryRepository struct {
	mu         sync.RWMutex
	executions map[int]*domain.JobExecution
	nextID     int
}

// NewInMemoryRepository creates a new in-memory job execution repository.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		executions: make(map[int]*domain.JobExecution),
		nextID:     1,
	}
}

func (r *InMemoryRepository) FindById(id int) (*domain.JobExecution, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	execution, ok := r.executions[id]
	if !ok {
		return nil, nil
	}
	copy := *execution
	return &copy, nil
}

func (r *InMemoryRepository) Save(execution *domain.JobExecution) error {
	if execution == nil {
		return fmt.Errorf("execution is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	if execution.Model == nil {
		execution.Model = &gorm.Model{}
	}
	execution.ID = uint(r.nextID)
	execution.CreatedAt = now
	execution.UpdatedAt = now
	r.executions[r.nextID] = cloneExecution(execution)
	r.nextID++
	return nil
}

func (r *InMemoryRepository) Update(execution *domain.JobExecution) error {
	if execution == nil {
		return fmt.Errorf("execution is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	id := int(execution.ID)
	if _, ok := r.executions[id]; !ok {
		return fmt.Errorf("execution not found")
	}
	execution.UpdatedAt = time.Now().UTC()
	r.executions[id] = cloneExecution(execution)
	return nil
}

func cloneExecution(in *domain.JobExecution) *domain.JobExecution {
	copy := *in
	if in.Model != nil {
		m := *in.Model
		copy.Model = &m
	}
	return &copy
}
