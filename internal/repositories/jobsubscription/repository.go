package jobsubscription

import "github.com/ethanbaker/assistant/internal/domain"

// Repository provides persistence APIs for job subscriptions.
type Repository interface {
	FindByClientAndJob(clientId, jobId int) (*domain.JobSubscription, error)
	FindActiveByJobId(jobId int) ([]*domain.JobSubscription, error)
	Save(sub *domain.JobSubscription) error
	Update(sub *domain.JobSubscription) error
}
