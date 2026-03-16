package outreach

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/ethanbaker/assistant/internal/domain"
	clientrepo "github.com/ethanbaker/assistant/internal/repositories/client"
	jobrepo "github.com/ethanbaker/assistant/internal/repositories/job"
	jobsubrepo "github.com/ethanbaker/assistant/internal/repositories/jobsubscription"
)

// JobService validates and persists job records.
type JobService struct {
	jobRepository jobrepo.Repository
}

func NewJobService(jobRepository jobrepo.Repository) *JobService {
	return &JobService{jobRepository: jobRepository}
}

// CreateJob creates a new job and persists it to the database
func (s *JobService) CreateJob(job *domain.Job) error {
	// Validate job
	if job == nil {
		return newServiceError(ErrorValidation, "job is required", nil)
	}
	if strings.TrimSpace(job.Name) == "" {
		return newServiceError(ErrorValidation, "job name is required", nil)
	}
	if err := validateScheduleJSON(job.Schedule); err != nil {
		return err
	}

	// Check for conflicting jobs
	existing, err := s.jobRepository.FindByName(job.Name)
	if err != nil {
		return newServiceError(ErrorInternal, "failed to check job uniqueness", err)
	}
	if existing != nil {
		return newServiceError(ErrorConflict, "job name already exists", nil)
	}

	// Save the new job
	if err := s.jobRepository.Save(job); err != nil {
		return newServiceError(ErrorInternal, "failed to save job", err)
	}
	return nil
}

// ClientService validates and persists outreach clients.
type ClientService struct {
	clientRepository clientrepo.Repository
}

func NewClientService(clientRepository clientrepo.Repository) *ClientService {
	return &ClientService{clientRepository: clientRepository}
}

// CreateClient creates a new client and persists it to the database
func (s *ClientService) CreateClient(name, webhookURL string) (*domain.Client, error) {
	// Validate parameters
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, newServiceError(ErrorValidation, "name is required", nil)
	}

	if _, err := url.ParseRequestURI(webhookURL); err != nil {
		return nil, newServiceError(ErrorValidation, "webhook_url is invalid", err)
	}

	// Check for conflicting clients
	existing, err := s.clientRepository.FindByName(name)
	if err != nil {
		return nil, newServiceError(ErrorInternal, "failed to check client uniqueness", err)
	}
	if existing != nil {
		return nil, newServiceError(ErrorConflict, "client name already exists", nil)
	}

	// Create an api key for the client
	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, newServiceError(ErrorInternal, "failed to generate api key", err)
	}

	// Create and save a new client
	client := &domain.Client{
		Name:       name,
		ApiKey:     apiKey,
		WebhookUrl: webhookURL,
	}

	if err := s.clientRepository.Save(client); err != nil {
		return nil, newServiceError(ErrorInternal, "failed to save client", err)
	}
	return client, nil
}

// Find a client by an existing api key
func (s *ClientService) FindByAPIKey(apiKey string) (*domain.Client, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, newServiceError(ErrorUnauthorized, "missing client api key", nil)
	}

	client, err := s.clientRepository.FindByApiKey(apiKey)
	if err != nil {
		return nil, newServiceError(ErrorInternal, "failed to lookup client", err)
	}
	if client == nil {
		return nil, newServiceError(ErrorUnauthorized, "invalid client api key", nil)
	}
	return client, nil
}

// SubscriptionService coordinates subscriptions using repository dependencies directly.
type SubscriptionService struct {
	clientRepository       clientrepo.Repository
	jobRepository          jobrepo.Repository
	subscriptionRepository jobsubrepo.Repository
}

func NewSubscriptionService(clientRepository clientrepo.Repository, jobRepository jobrepo.Repository, subscriptionRepository jobsubrepo.Repository) *SubscriptionService {
	return &SubscriptionService{
		clientRepository:       clientRepository,
		jobRepository:          jobRepository,
		subscriptionRepository: subscriptionRepository,
	}
}

// Subscribe creates a new subscription to an existing job
func (s *SubscriptionService) Subscribe(clientID int, jobName string, priority int) (*domain.JobSubscription, error) {
	// Validate parameters
	if priority <= 0 {
		return nil, newServiceError(ErrorValidation, "priority must be a positive integer", nil)
	}

	// Find existing job
	job, err := s.jobRepository.FindByName(strings.TrimSpace(jobName))
	if err != nil {
		return nil, newServiceError(ErrorInternal, "failed to find job", err)
	}
	if job == nil {
		return nil, newServiceError(ErrorNotFound, "job not found", nil)
	}

	// Find client creating subscription
	client, err := s.clientRepository.FindById(clientID)
	if err != nil {
		return nil, newServiceError(ErrorInternal, "failed to find client", err)
	}
	if client == nil {
		return nil, newServiceError(ErrorUnauthorized, "client not found", nil)
	}

	existing, err := s.subscriptionRepository.FindByClientAndJob(clientID, int(job.ID))
	if err != nil {
		return nil, newServiceError(ErrorInternal, "failed to find existing subscription", err)
	}

	// If subscription exists and the subscription is active, throw a conflict error
	if existing != nil && existing.Active {
		return nil, newServiceError(ErrorConflict, "client already has an active subscription for this job", nil)
	}

	// If subscription exists and the subscription is not active, reactivate subscription
	if existing != nil && !existing.Active {
		existing.Active = true
		existing.Priority = priority
		if err := s.subscriptionRepository.Update(existing); err != nil {
			return nil, newServiceError(ErrorInternal, "failed to reactivate subscription", err)
		}
		return existing, nil
	}

	// Create and save subscription
	sub := &domain.JobSubscription{
		Priority: priority,
		Active:   true,
		ClientId: clientID,
		JobId:    int(job.ID),
	}

	if err := s.subscriptionRepository.Save(sub); err != nil {
		return nil, newServiceError(ErrorInternal, "failed to save subscription", err)
	}
	return sub, nil
}

// Unsubscribe deactivates a subscription to a job from a client
func (s *SubscriptionService) Unsubscribe(clientID int, jobName string) error {
	// Check for job existence
	job, err := s.jobRepository.FindByName(strings.TrimSpace(jobName))
	if err != nil {
		return newServiceError(ErrorInternal, "failed to find job", err)
	}
	if job == nil {
		return newServiceError(ErrorNotFound, "job not found", nil)
	}

	// Check for existing subscription
	sub, err := s.subscriptionRepository.FindByClientAndJob(clientID, int(job.ID))
	if err != nil {
		return newServiceError(ErrorInternal, "failed to find subscription", err)
	}
	if sub == nil || !sub.Active {
		return newServiceError(ErrorNotFound, "active subscription not found", nil)
	}

	// Update subscription and persist
	sub.Active = false
	if err := s.subscriptionRepository.Update(sub); err != nil {
		return newServiceError(ErrorInternal, "failed to unsubscribe", err)
	}

	return nil
}

// Helper function to validate schedule json
func validateScheduleJSON(raw json.RawMessage) error {
	if len(raw) == 0 {
		return newServiceError(ErrorValidation, "schedule is required", nil)
	}

	// Make sure json has a type field with a valid schedule type
	var envelope domain.ScheduleEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return newServiceError(ErrorValidation, "schedule must be valid json", err)
	}

	// Convert json to specific schedule struct
	switch envelope.Type {
	case domain.ScheduleCron:
		var schedule domain.CronSchedule
		if err := json.Unmarshal(raw, &schedule); err != nil {
			return newServiceError(ErrorValidation, "invalid cron schedule payload", err)
		}
		if strings.TrimSpace(schedule.CronString) == "" {
			return newServiceError(ErrorValidation, "cron_string is required for cron schedule", nil)
		}
	case domain.ScheduleCustom:
		var schedule domain.CustomSchedule
		if err := json.Unmarshal(raw, &schedule); err != nil {
			return newServiceError(ErrorValidation, "invalid custom schedule payload", err)
		}
		if schedule.IntervalMs <= 0 {
			return newServiceError(ErrorValidation, "interval_ms must be > 0", nil)
		}
		if schedule.OffsetMs < 0 {
			return newServiceError(ErrorValidation, "offset_ms must be >= 0", nil)
		}
	default:
		return newServiceError(ErrorValidation, fmt.Sprintf("unsupported schedule type: %q", envelope.Type), nil)
	}

	return nil
}

// Helper method to generate an api key
func generateAPIKey() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
