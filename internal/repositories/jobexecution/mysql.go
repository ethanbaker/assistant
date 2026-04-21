package jobexecution

import (
	"fmt"

	"github.com/ethanbaker/assistant/internal/domain"
	"gorm.io/gorm"
)

// MySQLRepository stores job executions in MySQL.
type MySQLRepository struct {
	db *gorm.DB
}

// NewMySQLRepository creates a MySQL-backed job execution repository.
func NewMySQLRepository(db *gorm.DB) (*MySQLRepository, error) {
	repo := &MySQLRepository{db: db}
	if err := db.AutoMigrate(&domain.JobExecution{}); err != nil {
		return nil, fmt.Errorf("failed to migrate job execution table: %w", err)
	}
	return repo, nil
}

func (r *MySQLRepository) FindById(id int) (*domain.JobExecution, error) {
	var execution domain.JobExecution
	result := r.db.First(&execution, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find job execution by id: %w", result.Error)
	}
	return &execution, nil
}

func (r *MySQLRepository) Save(execution *domain.JobExecution) error {
	if err := r.db.Create(execution).Error; err != nil {
		return fmt.Errorf("failed to save job execution: %w", err)
	}
	return nil
}

func (r *MySQLRepository) Update(execution *domain.JobExecution) error {
	if err := r.db.Save(execution).Error; err != nil {
		return fmt.Errorf("failed to update job execution: %w", err)
	}
	return nil
}
