package outreach

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ethanbaker/assistant/pkg/utils"
	"github.com/nathan-osman/go-sunrise"
	"github.com/robfig/cron/v3"
)

// Manager handles scheduling and execution of outreach tasks
type Manager struct {
	// Implementations and tasks
	store       StoreInterface
	tasks       map[string]*Task
	responsesCh chan *Response
	cfg         *utils.Config

	// Concurrency
	mutex  sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc

	// Scheduling
	sunTicker *time.Ticker
	cron      *cron.Cron
	opts      *ManagerOptions
}

// ManagerOptions contains configuration options for the Manager
type ManagerOptions struct {
	Store StoreInterface `json:"-" yaml:"-"`

	Latitude  float64 `json:"latitude" yaml:"latitude"`   // Latitude for sunrise/sunset calculations
	Longitude float64 `json:"longitude" yaml:"longitude"` // Longitude for sunrise/sunset calculations
}

// NewManager creates a new outreach manager
func NewManager(cfg *utils.Config, opts *ManagerOptions) (*Manager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Add store
	var store StoreInterface
	if opts == nil || opts.Store == nil {
		cancel()
		return nil, fmt.Errorf("a valid store must be provided")
	}
	store = opts.Store

	// Create manager
	m := &Manager{
		store:       store,
		tasks:       make(map[string]*Task),
		responsesCh: make(chan *Response, 100), // Buffered channel
		mutex:       sync.RWMutex{},
		ctx:         ctx,
		cancel:      cancel,
		cron:        cron.New(),
		sunTicker:   time.NewTicker(1 * time.Minute),
		opts:        opts,
	}

	// Start the manager
	m.start()

	return m, nil
}

// start begins the manager's background operations
func (m *Manager) start() {
	m.cron.Start()

	// Start sunrise/sunset ticker in a goroutine
	go m.handleSunEvents()
}

// Stop gracefully stops the manager
func (m *Manager) Stop() {
	m.cancel()
	m.cron.Stop()
	m.sunTicker.Stop()
	close(m.responsesCh)
}

// GetResponseChannel returns the channel for receiving task responses
func (m *Manager) GetResponseChannel() <-chan *Response {
	return m.responsesCh
}

// RegisterImplementation registers a new implementation with the manager
func (m *Manager) RegisterImplementation(req *RegisterRequest) error {
	if req.ClientId == "" {
		return fmt.Errorf("client_id cannot be empty")
	}
	if req.CallbackUrl == "" {
		return fmt.Errorf("callback_url cannot be empty")
	}

	impl := &Implementation{
		ClientID:     req.ClientId,
		CallbackURL:  req.CallbackUrl,
		ClientSecret: req.ClientSecret,
		Active:       true,
	}

	return m.store.SaveImplementation(impl)
}

// UnregisterImplementation removes an implementation from the manager
func (m *Manager) UnregisterImplementation(clientID string) error {
	if clientID == "" {
		return fmt.Errorf("client_id cannot be empty")
	}

	return m.store.DisableImplementation(clientID)
}

// AuthenticateImplementation verifies client credentials
func (m *Manager) AuthenticateImplementation(clientID, clientSecret string) (*Implementation, error) {
	if clientID == "" {
		return nil, fmt.Errorf("client_id cannot be empty")
	}
	if clientSecret == "" {
		return nil, fmt.Errorf("client_secret cannot be empty")
	}

	return m.store.AuthenticateImplementation(clientID, clientSecret)
}

// LoadTasks registers a list of tasks with the manager
func (m *Manager) LoadTasks(tasks []*Task) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, task := range tasks {
		if err := m.loadTask(task); err != nil {
			return fmt.Errorf("failed to load task '%s': %w", task.Key, err)
		}
	}

	return nil
}

// loadTask registers a single task (called with mutex held)
func (m *Manager) loadTask(task *Task) error {
	if task.Key == "" {
		return fmt.Errorf("task key cannot be empty")
	}

	// Store the task
	m.tasks[task.Key] = task

	switch task.Cadence {
	case CronCadence: // Cron-based tasks are added to the cron instance
		return m.loadCronTask(task)

	case SunriseCadence, SunsetCadence: // Sunrise/Sunset tasks get added automatically; logic for running is handled in ticker
		if m.opts == nil {
			return fmt.Errorf("latitude and longitude must be set for sunrise/sunset tasks")
		}
		m.tasks[task.Key] = task
		return nil

	default:
		return fmt.Errorf("unsupported cadence type: %s", task.Cadence)
	}
}

// loadCronTask schedules a cron-based task
func (m *Manager) loadCronTask(task *Task) error {
	cronSpec, ok := task.CadenceParams["spec"].(string)
	if !ok || cronSpec == "" {
		return fmt.Errorf("cron tasks require 'spec' parameter")
	}

	_, err := m.cron.AddFunc(cronSpec, func() {
		m.executeTask(task)
	})

	return err
}

// executeTask runs a task and sends the response to the channel
func (m *Manager) executeTask(task *Task) {
	//log.Printf("[OUTREACH]: Executing task: %s", task.Key)

	// Get client IDs from task params
	if len(task.ClientIds) == 0 {
		log.Printf("[OUTREACH]: Task '%s' has no client_ids specified", task.Key)
		return
	}

	// Convert to string slice and validate implementations exist
	var clients []struct {
		Id          string `json:"id"`
		CallbackUrl string `json:"callback_url"`
	}

	for _, clientId := range task.ClientIds {
		// Get implementation details
		impl, err := m.store.GetImplementation(clientId)
		if err != nil {
			log.Printf("[OUTREACH]: Implementation '%s' not found for task '%s': %v", clientId, task.Key, err)
			continue
		}

		if impl == nil || !impl.Active {
			log.Printf("[OUTREACH]: Implementation '%s' is inactive for task '%s'", clientId, task.Key)
			continue
		}

		// Add to clients list
		clients = append(clients, struct {
			Id          string `json:"id"`
			CallbackUrl string `json:"callback_url"`
		}{
			Id:          impl.ClientID,
			CallbackUrl: impl.CallbackURL,
		})
	}

	// If no valid clients, skip
	if len(clients) == 0 {
		log.Printf("[OUTREACH]: No valid clients found for task '%s'", task.Key)
		return
	}

	// Run the task's function if defined
	if task.Run == nil {
		log.Printf("[OUTREACH]: Task '%s' has no run function defined", task.Key)
		return
	}
	output := task.Run(m.cfg)

	// If output is nil, skip
	if output == nil {
		//log.Printf("[OUTREACH]: Task '%s' returned nil output", task.Key)
		return
	}

	// Create response
	response := &Response{
		Status:        "success",
		Key:           task.Key,
		IdempotencyId: fmt.Sprintf("%s-%d", task.Key, time.Now().UnixNano()),
		Params:        task.Params,
		Clients:       clients,
		Content:       fmt.Sprintf("%v", output.Content),
		Data:          output.Data,
	}

	// Send to response channel
	select {
	case m.responsesCh <- response:
		log.Printf("[OUTREACH]: Task '%s' response sent to channel", task.Key)
	case <-m.ctx.Done():
		log.Printf("[OUTREACH]: Manager context cancelled, dropping task '%s' response", task.Key)
	default:
		log.Printf("[OUTREACH]: Response channel full, dropping task '%s' response", task.Key)
	}
}

// handleSunEvents processes sunrise and sunset events
func (m *Manager) handleSunEvents() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-m.sunTicker.C:
			m.checkSunEvents()
		}
	}
}

// checkSunEvents checks if it's time for sunrise or sunset tasks
func (m *Manager) checkSunEvents() {
	now := time.Now()

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Find sunrise/sunset
	lat, long := m.opts.Latitude, m.opts.Longitude
	year, month, day := now.Date()
	sunriseTime, sunsetTime := sunrise.SunriseSunset(lat, long, year, month, day)

	// Check if we're within a minute of sunrise (6:00 AM for simplicity)
	if m.isTimeForEvent(now, sunriseTime) {
		for _, task := range m.tasks {
			if task.Cadence == SunriseCadence {
				go m.executeTask(task)
			}
		}
	}

	// Check if we're within a minute of sunset (6:00 PM for simplicity)
	if m.isTimeForEvent(now, sunsetTime) {
		for _, task := range m.tasks {
			if task.Cadence == SunsetCadence {
				go m.executeTask(task)
			}
		}
	}
}

// isTimeForEvent checks if the current time matches the target hour and minute
func (m *Manager) isTimeForEvent(now time.Time, target time.Time) bool {
	return now.Hour() == target.Hour() && now.Minute() == target.Minute()
}

// GetImplementations returns all registered implementations
func (m *Manager) GetImplementations() []*Implementation {
	return m.store.ListImplementations()
}

// GetTasks returns all loaded tasks
func (m *Manager) GetTasks() []*Task {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	tasks := make([]*Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		tasks = append(tasks, task)
	}

	return tasks
}
