package client

import (
	"fmt"

	"github.com/ethanbaker/assistant/internal/domain"
	"gorm.io/gorm"
)

// MySQLRepository stores clients in MySQL.
type MySQLRepository struct {
	db *gorm.DB
}

// NewMySQLRepository creates a MySQL-backed client repository.
func NewMySQLRepository(db *gorm.DB) (*MySQLRepository, error) {
	repo := &MySQLRepository{db: db}
	if err := db.AutoMigrate(&domain.Client{}); err != nil {
		return nil, fmt.Errorf("failed to migrate client table: %w", err)
	}
	return repo, nil
}

func (r *MySQLRepository) FindById(id int) (*domain.Client, error) {
	var client domain.Client
	result := r.db.First(&client, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find client by id: %w", result.Error)
	}
	return &client, nil
}

func (r *MySQLRepository) FindByApiKey(apiKey string) (*domain.Client, error) {
	var client domain.Client
	result := r.db.Where("api_key = ?", apiKey).First(&client)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find client by api key: %w", result.Error)
	}
	return &client, nil
}

func (r *MySQLRepository) FindByName(name string) (*domain.Client, error) {
	var client domain.Client
	result := r.db.Where("name = ?", name).First(&client)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find client by name: %w", result.Error)
	}
	return &client, nil
}

func (r *MySQLRepository) Save(client *domain.Client) error {
	if err := r.db.Create(client).Error; err != nil {
		return fmt.Errorf("failed to save client: %w", err)
	}
	return nil
}
