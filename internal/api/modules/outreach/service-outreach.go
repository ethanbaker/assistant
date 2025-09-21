package outreach_module

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	outreach_dailydigest "github.com/ethanbaker/assistant/internal/outreaches/daily-digest"
	outreach_notionschedule "github.com/ethanbaker/assistant/internal/outreaches/notion-schedule"
	outreach_store "github.com/ethanbaker/assistant/internal/stores/outreach"
	"github.com/ethanbaker/assistant/pkg/outreach"
	"github.com/ethanbaker/assistant/pkg/sdk"
	"github.com/ethanbaker/assistant/pkg/utils"
	"github.com/go-sql-driver/mysql"
	"gopkg.in/yaml.v3"
)

const MAX_SEND_OUTREACH_RETRIES = 3

// OutreachService handles outreach operations and manages the outreach manager
type OutreachService struct {
	manager    *outreach.Manager
	httpClient *http.Client
	ctx        context.Context
	cancel     context.CancelFunc
	mutex      sync.RWMutex
}

var outreachService *OutreachService

// outreachTaskFunctions maps task keys to their corresponding run functions
var outreachTaskFunctions = map[string]outreach.TaskRunFunction{
	"daily-digest":    outreach_dailydigest.CreateDailyDigest,
	"notion-schedule": outreach_notionschedule.NotionScheduleReminder,
}

// outreachInits contains a list of outreach initialization functions
var outreachInits = map[string]func(cfg *utils.Config) error{
	"daily-digest":    outreach_dailydigest.Init,
	"notion-schedule": outreach_notionschedule.Init,
}

/** ---- INIT ---- */

// Init creates a new outreach service
func Init(cfg *utils.Config) error {
	var err error

	// Load manager config
	cfgPath := cfg.Get("OUTREACH_CONFIG_PATH")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return fmt.Errorf("outreach config file not found at %s", cfgPath)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to read outreach config file: %w", err)
	}

	var opts outreach.ManagerOptions
	if err := yaml.Unmarshal(data, &opts); err != nil {
		return fmt.Errorf("failed to parse outreach config file: %w", err)
	}

	// Create MySQL config
	dbConfig := mysql.Config{
		User:      cfg.Get("MYSQL_USER"),
		Passwd:    cfg.Get("MYSQL_ROOT_PASSWORD"),
		Net:       "tcp",
		Addr:      fmt.Sprintf("%s:%s", cfg.Get("MYSQL_HOST"), cfg.Get("MYSQL_PORT")),
		DBName:    cfg.Get("MYSQL_DATABASE"),
		ParseTime: true,
	}

	// Create store
	var store outreach.StoreInterface
	if dbConfig.DBName != "" {
		// Create sql store
		if store, err = outreach_store.NewStore(dbConfig.FormatDSN()); err != nil {
			return err
		}
	} else {
		// Fallback to in-memory store
		log.Println("[OUTREACH]: Warning, MYSQL_DATABASE not set, using in-memory store (data will not persist across restarts)")
		store = outreach_store.NewInMemoryStore()
	}

	opts.Store = store

	// Create manager
	manager, err := outreach.NewManager(cfg, &opts)
	if err != nil {
		return err
	}

	// Create database connection for service
	ctx, cancel := context.WithCancel(context.Background())
	service := &OutreachService{
		manager:    manager,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		ctx:        ctx,
		cancel:     cancel,
		mutex:      sync.RWMutex{},
	}

	// Start response listener
	go service.listenForResponses()

	taskPath := cfg.Get("OUTREACH_TASKS_PATH")

	// Load tasks on startup
	if err := service.loadTasksFromConfig(taskPath); err != nil {
		return fmt.Errorf("failed to load tasks from config: %w", err)
	}

	// Run outreach inits
	for name, initFunc := range outreachInits {
		if err := initFunc(cfg); err != nil {
			return fmt.Errorf("failed to run outreach init for %s: %w", name, err)
		}
	}

	outreachService = service
	return nil
}

// listenForResponses is a helper function that listens for responses from the manager and forwards them to implementations
func (s *OutreachService) listenForResponses() {
	responseCh := s.manager.GetResponseChannel()

	for {
		select {
		case <-s.ctx.Done():
			return
		case response, ok := <-responseCh:
			if !ok {
				log.Println("[OUTREACH]: Response channel closed")
				return
			}

			// Forward response to each client
			sent := false
			for i := 0; i < len(response.Clients) && !sent; i++ {
				client := response.Clients[i]

				for range MAX_SEND_OUTREACH_RETRIES {
					// Send response and log any errors
					err := s.forwardResponseToClient(client.CallbackUrl, response)
					if err == nil {
						sent = true
						break
					}

					log.Printf("[OUTREACH]: Failed to forward response to %s: %v", client.CallbackUrl, err)
				}
			}
		}
	}
}

// loadTasksFromConfig is a helper function to loads tasks from the configuration file
func (s *OutreachService) loadTasksFromConfig(path string) error {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("[OUTREACH]: Tasks configuration file not found at %s, skipping task loading", path)
		return nil
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read tasks configuration file: %w", err)
	}

	// Parse YAML
	var config struct {
		Tasks []*outreach.Task `yaml:"tasks"`
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse tasks configuration file: %w", err)
	}

	// Map tasks to run functions
	for _, task := range config.Tasks {
		if runFunc, exists := outreachTaskFunctions[task.Key]; exists {
			task.Run = runFunc
		} else {
			return fmt.Errorf("no run function found for task key '%s'", task.Key)
		}
	}

	// Load tasks
	if len(config.Tasks) == 0 {
		return fmt.Errorf("no tasks found in configuration file")
	}

	if err := s.manager.LoadTasks(config.Tasks); err != nil {
		return fmt.Errorf("failed to load tasks from configuration: %w", err)
	}

	log.Printf("[OUTREACH]: Successfully loaded %d tasks from configuration", len(config.Tasks))
	return nil

}

/** ---- SERVICE METHODS ---- */

// Stop gracefully stops the service
func (s *OutreachService) Stop() {
	s.cancel()
	s.manager.Stop()
}

// RegisterImplementation registers a new implementation
func (s *OutreachService) RegisterImplementation(req *sdk.OutreachRegisterRequest) error {
	// Validate client secret if provided
	if req.ClientSecret != "" {
		if _, err := s.AuthenticateClient(req.ClientId, req.ClientSecret); err != nil {
			return fmt.Errorf("invalid client credentials: %w", err)
		}
	}

	// Convert to outreach register request
	outreachReq := &outreach.RegisterRequest{
		ClientId:     req.ClientId,
		CallbackUrl:  req.CallbackUrl,
		ClientSecret: req.ClientSecret,
	}

	// Register with manager
	if err := s.manager.RegisterImplementation(outreachReq); err != nil {
		return fmt.Errorf("failed to register implementation: %w", err)
	}

	return nil
}

// UnregisterImplementation removes an implementation
func (s *OutreachService) UnregisterImplementation(clientId string) error {
	return s.manager.UnregisterImplementation(clientId)
}

// GetImplementations returns all registered implementations
func (s *OutreachService) GetImplementations() (*sdk.OutreachListImplementationsResponse, error) {
	implementations := s.manager.GetImplementations()

	var sdkImplementations []sdk.OutreachImplementation
	for _, impl := range implementations {
		sdkImplementations = append(sdkImplementations, sdk.OutreachImplementation{
			ClientId:    impl.ClientID,
			CallbackUrl: impl.CallbackURL,
		})
	}

	return &sdk.OutreachListImplementationsResponse{
		Implementations: sdkImplementations,
		Count:           len(sdkImplementations),
	}, nil
}

// GetStatus returns the current status of the outreach service
func (s *OutreachService) GetStatus() *sdk.OutreachStatusResponse {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	tasks := s.manager.GetTasks()
	implementations := s.manager.GetImplementations()

	return &sdk.OutreachStatusResponse{
		Status: "running",
		TasksStatus: sdk.OutreachTaskStatus{
			Loaded: len(tasks),
		},
		ImplementationsCount: len(implementations),
		ManagerRunning:       true,
	}
}

// AuthenticateClient authenticates a client using the store
func (s *OutreachService) AuthenticateClient(clientID, clientSecret string) (*outreach.Implementation, error) {
	impl, err := s.manager.AuthenticateImplementation(clientID, clientSecret)
	if err != nil {
		return nil, err
	}

	if !impl.Active {
		return nil, fmt.Errorf("implementation with client_id '%s' is inactive", clientID)
	}

	return impl, nil
}

/** ---- HELPERS ---- */

// forwardResponseToClient sends a response to a specific client implementation
func (s *OutreachService) forwardResponseToClient(callbackUrl string, response *outreach.Response) error {
	// Create outreach request
	outreachReq := &sdk.OutreachRequest{
		Id:      response.IdempotencyId,
		Key:     response.Key,
		Params:  response.Params,
		Content: response.Content,
		Data:    response.Data,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(outreachReq)
	if err != nil {
		return err
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(s.ctx, http.MethodPost, callbackUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "assistant-outreach/1.0")

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Log response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("outreach request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
