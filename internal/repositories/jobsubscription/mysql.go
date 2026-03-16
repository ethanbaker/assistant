package jobsubscription

import (
	"fmt"

	"github.com/ethanbaker/assistant/internal/domain"
	"gorm.io/gorm"
)

// MySQLRepository stores job subscriptions in MySQL.
type MySQLRepository struct {
	db *gorm.DB
}

// NewMySQLRepository creates a MySQL-backed subscription repository.
func NewMySQLRepository(db *gorm.DB) (*MySQLRepository, error) {
	repo := &MySQLRepository{db: db}
	if err := db.AutoMigrate(&domain.JobSubscription{}); err != nil {
		return nil, fmt.Errorf("failed to migrate job subscription table: %w", err)
	}
	return repo, nil
}

func (r *MySQLRepository) FindByClientAndJob(clientId, jobId int) (*domain.JobSubscription, error) {
	var sub domain.JobSubscription
	result := r.db.Where("client_id = ? AND job_id = ?", clientId, jobId).First(&sub)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find subscription by client and job: %w", result.Error)
	}
	return &sub, nil
}

func (r *MySQLRepository) FindActiveByJobId(jobId int) ([]*domain.JobSubscription, error) {
	var subs []*domain.JobSubscription
	if err := r.db.Where("job_id = ? AND active = ?", jobId, true).Order("priority ASC").Find(&subs).Error; err != nil {
		return nil, fmt.Errorf("failed to find active subscriptions by job id: %w", err)
	}
	return subs, nil
}

func (r *MySQLRepository) Save(sub *domain.JobSubscription) error {
	if err := r.db.Create(sub).Error; err != nil {
		return fmt.Errorf("failed to save subscription: %w", err)
	}
	return nil
}

func (r *MySQLRepository) Update(sub *domain.JobSubscription) error {
	if err := r.db.Save(sub).Error; err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}
	return nil
}
