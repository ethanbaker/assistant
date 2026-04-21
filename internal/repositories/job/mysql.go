package job

import (
	"fmt"

	"github.com/ethanbaker/assistant/internal/domain"
	"gorm.io/gorm"
)

// MySQLRepository stores jobs in MySQL.
type MySQLRepository struct {
	db *gorm.DB
}

// NewMySQLRepository creates a MySQL-backed job repository.
func NewMySQLRepository(db *gorm.DB) (*MySQLRepository, error) {
	repo := &MySQLRepository{db: db}
	if err := db.AutoMigrate(&domain.Job{}); err != nil {
		return nil, fmt.Errorf("failed to migrate job table: %w", err)
	}
	return repo, nil
}

func (r *MySQLRepository) FindById(id int) (*domain.Job, error) {
	var job domain.Job
	result := r.db.First(&job, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find job by id: %w", result.Error)
	}
	return &job, nil
}

func (r *MySQLRepository) FindByName(name string) (*domain.Job, error) {
	var job domain.Job
	result := r.db.Where("name = ?", name).First(&job)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find job by name: %w", result.Error)
	}
	return &job, nil
}

func (r *MySQLRepository) FindAllActive() ([]*domain.Job, error) {
	var jobs []*domain.Job
	if err := r.db.Where("active = ?", true).Find(&jobs).Error; err != nil {
		return nil, fmt.Errorf("failed to find active jobs: %w", err)
	}
	return jobs, nil
}

func (r *MySQLRepository) Save(job *domain.Job) error {
	if err := r.db.Create(job).Error; err != nil {
		return fmt.Errorf("failed to save job: %w", err)
	}
	return nil
}

func (r *MySQLRepository) Update(job *domain.Job) error {
	if err := r.db.Save(job).Error; err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}
	return nil
}
