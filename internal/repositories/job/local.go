package job

import (
	"fmt"
	"sync"
	"time"

	"github.com/ethanbaker/assistant/internal/domain"
	"gorm.io/gorm"
)

// InMemoryRepository stores jobs in memory.
type InMemoryRepository struct {
	mu     sync.RWMutex
	jobs   map[int]*domain.Job
	nextID int
}

// NewInMemoryRepository creates a new in-memory job repository.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		jobs:   make(map[int]*domain.Job),
		nextID: 1,
	}
}

func (r *InMemoryRepository) FindById(id int) (*domain.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	job, ok := r.jobs[id]
	if !ok {
		return nil, nil
	}
	copy := *job
	return &copy, nil
}

func (r *InMemoryRepository) FindByName(name string) (*domain.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, job := range r.jobs {
		if job.Name == name {
			copy := *job
			return &copy, nil
		}
	}
	return nil, nil
}

func (r *InMemoryRepository) FindAllActive() ([]*domain.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	jobs := make([]*domain.Job, 0)
	for _, job := range r.jobs {
		if job.Active {
			copy := *job
			jobs = append(jobs, &copy)
		}
	}
	return jobs, nil
}

func (r *InMemoryRepository) Save(job *domain.Job) error {
	if job == nil {
		return fmt.Errorf("job is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	if job.Model == nil {
		job.Model = &gorm.Model{}
	}
	job.ID = uint(r.nextID)
	job.CreatedAt = now
	job.UpdatedAt = now
	r.jobs[r.nextID] = cloneJob(job)
	r.nextID++
	return nil
}

func (r *InMemoryRepository) Update(job *domain.Job) error {
	if job == nil {
		return fmt.Errorf("job is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	id := int(job.ID)
	if _, ok := r.jobs[id]; !ok {
		return fmt.Errorf("job not found")
	}
	job.UpdatedAt = time.Now().UTC()
	r.jobs[id] = cloneJob(job)
	return nil
}

func cloneJob(in *domain.Job) *domain.Job {
	copy := *in
	if in.Model != nil {
		m := *in.Model
		copy.Model = &m
	}
	if in.Schedule != nil {
		copy.Schedule = append([]byte(nil), in.Schedule...)
	}
	if in.Parameters != nil {
		copy.Parameters = append([]byte(nil), in.Parameters...)
	}
	if in.Subscriptions != nil {
		copy.Subscriptions = append([]domain.JobSubscription(nil), in.Subscriptions...)
	}
	return &copy
}
