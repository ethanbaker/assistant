package jobexecution

import "github.com/ethanbaker/assistant/internal/domain"

// Repository provides persistence APIs for job executions.
type Repository interface {
	FindById(id int) (*domain.JobExecution, error)
	Save(execution *domain.JobExecution) error
	Update(execution *domain.JobExecution) error
}
