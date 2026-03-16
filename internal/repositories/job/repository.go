package job

import "github.com/ethanbaker/assistant/internal/domain"

// Repository provides persistence APIs for jobs.
type Repository interface {
	FindById(id int) (*domain.Job, error)
	FindByName(name string) (*domain.Job, error)
	FindAllActive() ([]*domain.Job, error)
	Save(job *domain.Job) error
	Update(job *domain.Job) error
}
