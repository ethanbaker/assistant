package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ethanbaker/assistant/pkg/config"
	"github.com/ethanbaker/assistant/pkg/logger"
	"github.com/ethanbaker/assistant/pkg/sdk"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/goccy/go-yaml"
)

const (
	outreachDefaultPort = "9000"
	outreachWebhook     = "/api/outreach-message"
)

// OutreachJob describes a single outreach job subscription with its priority.
type OutreachJob struct {
	Name     string `yaml:"name"`
	Priority int    `yaml:"priority"`
}

type outreachConfig struct {
	Jobs []OutreachJob `yaml:"jobs"`
}

// loadOutreachJobs reads outreach job definitions from a YAML file.
// The file path defaults to "outreach.yml" but can be overridden via the
// OUTREACH_CONFIG_FILE environment variable.
func loadOutreachJobs() ([]OutreachJob, error) {
	path := config.GetenvWithDefault("OUTREACH_CONFIG_FILE", "outreach.yml")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var cfg outreachConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	return cfg.Jobs, nil
}

// Outreach manager handles outreach tasks
type OutreachManager struct {
	api        *sdk.Client
	clientKey  string
	clientName string

	server *http.Server
	engine *gin.Engine
	host   string

	idempotencyStore map[string]bool // idempotencyStore tracks processed outreach message IDs to prevent duplicate delivery.

	jobs          []OutreachJob
	clientKeyRepo OutreachClientRepository
}

type OutreachManagerConfig struct {
	Jobs          []OutreachJob
	SDKClient     *sdk.Client
	ClientName    string
	Host          string
	Port          string
	ClientKeyRepo OutreachClientRepository
}

// NewOutreachManager creates a new outreach manager from the provided config
func NewOutreachManager(cfg OutreachManagerConfig) (*OutreachManager, error) {
	// Validate config
	if cfg.SDKClient == nil {
		return nil, fmt.Errorf("sdk client cannot be nil")
	}
	if cfg.ClientKeyRepo == nil {
		return nil, fmt.Errorf("client key repository cannot be nil")
	}
	clientName := strings.TrimSpace(cfg.ClientName)
	if clientName == "" {
		return nil, fmt.Errorf("client name is required")
	}
	host := strings.TrimSpace(cfg.Host)
	if host == "" {
		return nil, fmt.Errorf("host is required")
	}
	port := strings.TrimSpace(cfg.Port)
	if port == "" {
		port = outreachDefaultPort
	}

	m := &OutreachManager{
		api:              cfg.SDKClient,
		jobs:             cfg.Jobs,
		clientName:       clientName,
		host:             host,
		idempotencyStore: make(map[string]bool),
		clientKeyRepo:    cfg.ClientKeyRepo,
	}

	// Load or register the client key
	if key, ok := cfg.ClientKeyRepo.Get(); ok {
		m.clientKey = key
		logger.Info("loaded outreach client key from repository")
	} else {
		if err := m.register(); err != nil {
			return nil, fmt.Errorf("register outreach client: %w", err)
		}
	}

	// Create gin api
	allowedOrigins := config.GetenvWithDefault("CORS_ALLOWED_ORIGINS", "*")

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.SetTrustedProxies(nil)

	engine.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Split(allowedOrigins, ","),
		AllowMethods:     []string{"OPTIONS", "GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	engine.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, sdk.NewErrorResponse(http.StatusNotFound, "NOT_FOUND", "route not found"))
	})

	m.engine = engine
	m.server = &http.Server{
		Addr:    ":" + port,
		Handler: engine,
	}

	return m, nil
}

// addHandler adds an external handler to the internal outreach api
func (m *OutreachManager) addHandler(path string, handler func(c *gin.Context, payload sdk.WebhookPayload) error) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("path is required")
	}
	if handler == nil {
		return fmt.Errorf("handler func is required")
	}

	m.engine.POST(path, func(c *gin.Context) {
		var req sdk.WebhookPayload
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(sdk.NewErrorResponse(http.StatusBadRequest, "BAD_REQUEST", "could not parse request body").AsGinResponse())
			return
		}

		// Idempotency check
		id := fmt.Sprint(req.ID)
		if complete, ok := m.idempotencyStore[id]; ok && complete {
			c.JSON(sdk.NewSuccessMessage("outreach message already sent").AsGinResponse())
			return
		}
		m.idempotencyStore[id] = false

		// Delegate to handler
		if err := handler(c, req); err != nil {
			c.JSON(sdk.NewInternalServerError(fmt.Sprintf("handler failed: %v", err)).AsGinResponse())
			return
		}
		m.idempotencyStore[id] = true
	})
	return nil
}

// register as an outreach client
func (m *OutreachManager) register() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := m.api.RegisterClient(ctx, &sdk.RegisterClientRequest{
		Name:       m.clientName,
		WebhookURL: m.host + outreachWebhook,
	})
	if err != nil {
		return fmt.Errorf("RegisterClient: %w", err)
	}

	m.clientKey = resp.APIKey
	logger.Infof("registered outreach client (id=%d)", resp.ClientID)

	if err := m.clientKeyRepo.Save(m.clientKey); err != nil {
		logger.Errorf("failed to persist outreach client key: %v", err)
	}

	return nil
}

// subscribe subscribes to all configured jobs
func (m *OutreachManager) subscribe() error {
	ctx := context.Background()
	//ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	//defer cancel()

	if len(m.jobs) == 0 {
		logger.Warn("outreach has no active jobs")
	}

	for _, job := range m.jobs {
		subResp, err := m.api.Subscribe(ctx, m.clientKey, &sdk.SubscribeRequest{
			JobName:  job.Name,
			Priority: job.Priority,
		})
		if err != nil {
			return fmt.Errorf("Subscribe(%q): %w", job.Name, err)
		}
		logger.Infof("subscribed to outreach job %q (subscription_id=%d)", job.Name, subResp.SubscriptionID)
	}

	return nil
}

// unsubscribes this bot from all configured outreach jobs.
func (m *OutreachManager) unsubscribe() error {
	if m.clientKey == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var errs []string
	for _, job := range m.jobs {
		jobName := strings.TrimSpace(job.Name)
		if jobName == "" {
			continue
		}
		if err := m.api.Unsubscribe(ctx, m.clientKey, &sdk.UnsubscribeRequest{JobName: jobName}); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", jobName, err))
		} else {
			logger.Infof("unsubscribed from outreach job %q", jobName)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("unsubscribe errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

// start starts the internal HTTP server that receives outreach webhook calls
func (m *OutreachManager) start() {
	logger.Infof("outreach server listening on %s", m.server.Addr)
	go m.server.ListenAndServe()
}

// stop stops the internal HTTP server
func (m *OutreachManager) stop(ctx context.Context) error {
	logger.Info("unsubscribing from outreach...")
	if err := m.unsubscribe(); err != nil {
		logger.Errorf("error during outreach unsubscribing: %v", err)
	}

	logger.Info("shutting down server")
	return m.server.Shutdown(ctx)
}
