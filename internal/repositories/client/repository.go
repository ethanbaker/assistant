package client

import "github.com/ethanbaker/assistant/internal/domain"

// Repository provides persistence APIs for outreach clients.
type Repository interface {
	FindById(id int) (*domain.Client, error)
	FindByApiKey(apiKey string) (*domain.Client, error)
	FindByName(name string) (*domain.Client, error)
	Save(client *domain.Client) error
}
