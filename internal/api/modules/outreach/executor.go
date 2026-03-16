package outreach

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethanbaker/assistant/internal/config"
	"github.com/ethanbaker/assistant/internal/domain"
	"github.com/ethanbaker/assistant/internal/logger"
	clientrepo "github.com/ethanbaker/assistant/internal/repositories/client"
	jobrepo "github.com/ethanbaker/assistant/internal/repositories/job"
	jobexecutionrepo "github.com/ethanbaker/assistant/internal/repositories/jobexecution"
	jobsubrepo "github.com/ethanbaker/assistant/internal/repositories/jobsubscription"
)

// HandlerFunc executes the configured job handler using serialized parameters.
type HandlerFunc func(ctx context.Context, params json.RawMessage) (string, error)

// scheduleRunner interface contains a function that waits until it's due to run
type scheduleRunner interface {
	WaitUntilDue(ctx context.Context) error
}

type cronScheduleRunner struct {
	parsed *simpleCron
}

func (r *cronScheduleRunner) WaitUntilDue(ctx context.Context) error {
	next, err := r.parsed.Next(time.Now())
	if err != nil {
		return err
	}

	wait := time.Until(next)
	if wait <= 0 {
		return nil
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type customScheduleRunner struct {
	interval time.Duration
	offset   time.Duration
	ticker   *time.Ticker
	started  bool
}

func (r *customScheduleRunner) WaitUntilDue(ctx context.Context) error {
	if !r.started {
		r.started = true
		if r.offset > 0 {
			timer := time.NewTimer(r.offset)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
			}
		}
		r.ticker = time.NewTicker(r.interval)
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-r.ticker.C:
		return nil
	}
}

// runningJob represents a job running in the current context
type runningJob struct {
	job      *domain.Job
	schedule scheduleRunner
	cancel   context.CancelFunc
}

// ExecutorConfig defines dependencies and runtime settings for the outreach executor.
type ExecutorConfig struct {
	JobRepository          jobrepo.Repository
	ClientRepository       clientrepo.Repository
	SubscriptionRepository jobsubrepo.Repository
	ExecutionRepository    jobexecutionrepo.Repository
	Handlers               map[string]HandlerFunc
	HTTPClient             *http.Client
	PollInterval           time.Duration
	DeliveryTimeout        time.Duration
}

// Executor runs outreach jobs in the background.
type Executor struct {
	jobRepository          jobrepo.Repository
	clientRepository       clientrepo.Repository
	subscriptionRepository jobsubrepo.Repository
	executionRepository    jobexecutionrepo.Repository
	handlers               map[string]HandlerFunc
	httpClient             *http.Client
	pollInterval           time.Duration
	deliveryTimeout        time.Duration

	mu      sync.RWMutex
	running map[int]*runningJob

	rootCtx    context.Context
	rootCancel context.CancelFunc
	wg         sync.WaitGroup
}

func NewExecutor(cfg ExecutorConfig) *Executor {
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	pollInterval := cfg.PollInterval
	if pollInterval <= 0 {
		pollInterval = parseSecondsEnv("OUTREACH_POLL_INTERVAL_SECONDS", 30)
	}

	deliveryTimeout := cfg.DeliveryTimeout
	if deliveryTimeout <= 0 {
		deliveryTimeout = parseSecondsEnv("OUTREACH_DELIVERY_TIMEOUT_SECONDS", 10)
	}

	return &Executor{
		jobRepository:          cfg.JobRepository,
		clientRepository:       cfg.ClientRepository,
		subscriptionRepository: cfg.SubscriptionRepository,
		executionRepository:    cfg.ExecutionRepository,
		handlers:               cfg.Handlers,
		httpClient:             httpClient,
		pollInterval:           pollInterval,
		deliveryTimeout:        deliveryTimeout,
		running:                make(map[int]*runningJob),
	}
}

// Start loads active jobs and launches the polling loop.
func (e *Executor) Start(ctx context.Context) error {
	e.rootCtx, e.rootCancel = context.WithCancel(ctx)
	if err := e.loadJobs(e.rootCtx); err != nil {
		return err
	}

	e.wg.Go(func() {
		e.listenForUpdates(e.rootCtx)
	})

	return nil
}

// Stop cancels active jobs and blocks until all goroutines exit.
func (e *Executor) Stop() {
	if e.rootCancel != nil {
		e.rootCancel()
	}

	e.mu.Lock()
	for id, run := range e.running {
		run.cancel()
		delete(e.running, id)
	}
	e.mu.Unlock()

	e.wg.Wait()
}

// loadJobs is a helper method that loads jobs from the job repository
func (e *Executor) loadJobs(ctx context.Context) error {
	jobs, err := e.jobRepository.FindAllActive()
	if err != nil {
		return fmt.Errorf("failed to load active jobs: %w", err)
	}

	for _, job := range jobs {
		runner, parseErr := parseScheduleRunner(job.Schedule)
		if parseErr != nil {
			logger.Warnf("Skipping outreach job %d (%s): invalid schedule: %v", job.ID, job.Name, parseErr)
			continue
		}
		e.startJob(ctx, job, runner)
	}

	return nil
}

// listenForUpdates is a helper function that listens for changes in the job source
func (e *Executor) listenForUpdates(ctx context.Context) {
	ticker := time.NewTicker(e.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.reconcileJobs(ctx)
		}
	}
}

// reconcileJobs is a helper function that checks for job updates and modifies existing schedule runners
func (e *Executor) reconcileJobs(ctx context.Context) {
	jobs, err := e.jobRepository.FindAllActive()
	if err != nil {
		logger.Errorf("Outreach executor poll failed: %v", err)
		return
	}

	seen := make(map[int]*domain.Job)
	for _, job := range jobs {
		seen[int(job.ID)] = job
	}

	e.mu.Lock()
	for id, run := range e.running {
		if _, ok := seen[id]; !ok {
			run.cancel()
			delete(e.running, id)
		}
	}
	e.mu.Unlock()

	for _, job := range jobs {
		id := int(job.ID)

		e.mu.RLock()
		existing, ok := e.running[id]
		e.mu.RUnlock()

		if !ok {
			runner, parseErr := parseScheduleRunner(job.Schedule)
			if parseErr != nil {
				logger.Warnf("Skipping new outreach job %d (%s): invalid schedule: %v", job.ID, job.Name, parseErr)
				continue
			}
			e.startJob(ctx, job, runner)
			continue
		}

		if !sameUpdatedAt(existing.job, job) {
			existing.cancel()
			runner, parseErr := parseScheduleRunner(job.Schedule)
			if parseErr != nil {
				logger.Warnf("Skipping updated outreach job %d (%s): invalid schedule: %v", job.ID, job.Name, parseErr)
				e.mu.Lock()
				delete(e.running, id)
				e.mu.Unlock()
				continue
			}
			e.startJob(ctx, job, runner)
		}
	}
}

// startJob is a helper function that starts a job with a given schedule runner
func (e *Executor) startJob(ctx context.Context, job *domain.Job, runner scheduleRunner) {
	jobCtx, cancel := context.WithCancel(ctx)
	run := &runningJob{job: job, schedule: runner, cancel: cancel}

	e.mu.Lock()
	e.running[int(job.ID)] = run
	e.mu.Unlock()

	e.wg.Go(func() {
		e.execute(jobCtx, run)
	})
}

// execute is a helper function that processes a running job infinitely
func (e *Executor) execute(ctx context.Context, run *runningJob) {
	for {
		if err := run.schedule.WaitUntilDue(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Warnf("WaitUntilDue failed for job %d: %v", run.job.ID, err)
			continue
		}

		e.processJob(ctx, run.job)
	}
}

// processJob is a helper function that processes a job for an output
func (e *Executor) processJob(ctx context.Context, job *domain.Job) {
	// Create a job execution model
	execution := &domain.JobExecution{
		Status: domain.Running,
		JobId:  int(job.ID),
	}

	if err := e.executionRepository.Save(execution); err != nil {
		logger.Errorf("Failed to save job execution for job %d: %v", job.ID, err)
		return
	}

	// Find the associated handler for the job
	handler, ok := e.handlers[job.Handler]
	if !ok {
		execution.Status = domain.JobFailed
		execution.Error = fmt.Sprintf("handler not found: %s", job.Handler)
		_ = e.executionRepository.Update(execution)
		return
	}

	// Run the handler for an output
	output, handlerErr := handler(ctx, job.Parameters)
	if handlerErr != nil {
		execution.Status = domain.JobFailed
		execution.Error = handlerErr.Error()
		_ = e.executionRepository.Update(execution)
		return
	}

	// Update the execution record
	execution.Output = output
	if err := e.executionRepository.Update(execution); err != nil {
		logger.Warnf("Failed to persist job execution output for job %d: %v", job.ID, err)
	}

	// Find subscriptions to the job
	subs, err := e.subscriptionRepository.FindActiveByJobId(int(job.ID))
	if err != nil {
		execution.Status = domain.SendingFailed
		execution.Error = fmt.Sprintf("failed to load subscriptions: %v", err)
		_ = e.executionRepository.Update(execution)
		return
	}

	if len(subs) == 0 {
		execution.Status = domain.SendingFailed
		execution.Error = "no active subscriptions"
		_ = e.executionRepository.Update(execution)
		return
	}

	// Sort subscriptions by priority
	for i := 1; i < len(subs); i++ {
		s := subs[i]
		j := i - 1

		for j >= 0 && s.Priority < subs[j].Priority {
			subs[j+1] = subs[j]
			j--
		}

		subs[j+1] = s
	}

	// Post to the subscriptions (stop on first success)
	for _, sub := range subs {
		now := time.Now().UTC()
		sub.LastAttemptedAt = &now
		_ = e.subscriptionRepository.Update(sub)

		// Find client from subscription
		client, clientErr := e.clientRepository.FindById(sub.ClientId)
		if clientErr != nil || client == nil {
			execution.Status = domain.SendingFailed
			execution.Error = fmt.Sprintf("failed to load client %d: %v", sub.ClientId, clientErr)
			_ = e.executionRepository.Update(execution)
			continue
		}

		// Send execution to webhook
		if sendErr := e.sendWebhook(ctx, client.WebhookUrl, execution); sendErr != nil {
			execution.Status = domain.SendingFailed
			execution.Error = sendErr.Error()
			_ = e.executionRepository.Update(execution)
			continue
		}

		// Update to success status
		sub.LastSuccessAt = &now
		_ = e.subscriptionRepository.Update(sub)

		execution.ClientId = sub.ClientId
		execution.Status = domain.SuccessfullySent
		execution.Error = ""
		_ = e.executionRepository.Update(execution)
		return
	}
}

// sendWebhook is a helper method to send a webhook to a client
func (e *Executor) sendWebhook(ctx context.Context, webhookURL string, execution *domain.JobExecution) error {
	// Create payload from job execution record
	payload := map[string]any{
		"job_execution_ids": []int{int(execution.ID)},
		"execution":         execution,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to serialize webhook payload: %w", err)
	}

	deliveryCtx, cancel := context.WithTimeout(ctx, e.deliveryTimeout)
	defer cancel()

	// Perform request
	req, err := http.NewRequestWithContext(deliveryCtx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook response status %d", resp.StatusCode)
	}

	return nil
}

// parseScheduleRunner is a helper function that parses json to a schedule runner
func parseScheduleRunner(raw json.RawMessage) (scheduleRunner, error) {
	var envelope domain.ScheduleEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("invalid schedule json: %w", err)
	}

	switch envelope.Type {
	case domain.ScheduleCron:
		var sched domain.CronSchedule
		if err := json.Unmarshal(raw, &sched); err != nil {
			return nil, fmt.Errorf("invalid cron schedule: %w", err)
		}
		parsed, err := parseSimpleCron(strings.TrimSpace(sched.CronString))
		if err != nil {
			return nil, fmt.Errorf("failed to parse cron expression: %w", err)
		}
		return &cronScheduleRunner{parsed: parsed}, nil
	case domain.ScheduleCustom:
		var sched domain.CustomSchedule
		if err := json.Unmarshal(raw, &sched); err != nil {
			return nil, fmt.Errorf("invalid custom schedule: %w", err)
		}
		if sched.IntervalMs <= 0 {
			return nil, fmt.Errorf("interval_ms must be > 0")
		}
		if sched.OffsetMs < 0 {
			return nil, fmt.Errorf("offset_ms must be >= 0")
		}
		return &customScheduleRunner{
			interval: time.Duration(sched.IntervalMs) * time.Millisecond,
			offset:   time.Duration(sched.OffsetMs) * time.Millisecond,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported schedule type: %s", envelope.Type)
	}
}

// Helper function to parse seconds from an environment
func parseSecondsEnv(key string, fallback int) time.Duration {
	raw := strings.TrimSpace(config.GetenvWithDefault(key, strconv.Itoa(fallback)))
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		value = fallback
	}
	return time.Duration(value) * time.Second
}

// Helper function to check for jobs updated at the same time. This is how new jobs are distinguished
func sameUpdatedAt(a, b *domain.Job) bool {
	return a != nil && b != nil && a.UpdatedAt.Equal(b.UpdatedAt)
}

// Cron parser

type cronField struct {
	any     bool
	allowed map[int]bool
}

type simpleCron struct {
	minute cronField
	hour   cronField
	dom    cronField
	month  cronField
	dow    cronField
}

func parseSimpleCron(spec string) (*simpleCron, error) {
	parts := strings.Fields(spec)
	if len(parts) != 5 {
		return nil, fmt.Errorf("expected 5 fields, got %d", len(parts))
	}

	minute, err := parseCronField(parts[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("minute field: %w", err)
	}
	hour, err := parseCronField(parts[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("hour field: %w", err)
	}
	dom, err := parseCronField(parts[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("day-of-month field: %w", err)
	}
	month, err := parseCronField(parts[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("month field: %w", err)
	}
	dow, err := parseCronField(parts[4], 0, 6)
	if err != nil {
		return nil, fmt.Errorf("day-of-week field: %w", err)
	}

	return &simpleCron{minute: minute, hour: hour, dom: dom, month: month, dow: dow}, nil
}

func (c *simpleCron) Next(from time.Time) (time.Time, error) {
	t := from.Truncate(time.Minute).Add(time.Minute)
	max := t.AddDate(1, 0, 0)

	for !t.After(max) {
		if c.matches(t) {
			return t, nil
		}
		t = t.Add(time.Minute)
	}

	return time.Time{}, fmt.Errorf("no execution time found within one year")
}

func (c *simpleCron) matches(t time.Time) bool {
	return fieldMatches(c.minute, t.Minute()) &&
		fieldMatches(c.hour, t.Hour()) &&
		fieldMatches(c.dom, t.Day()) &&
		fieldMatches(c.month, int(t.Month())) &&
		fieldMatches(c.dow, int(t.Weekday()))
}

func fieldMatches(field cronField, value int) bool {
	if field.any {
		return true
	}
	return field.allowed[value]
}

func parseCronField(raw string, min int, max int) (cronField, error) {
	if raw == "*" {
		return cronField{any: true}, nil
	}

	allowed := make(map[int]bool)
	segments := strings.Split(raw, ",")
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			return cronField{}, fmt.Errorf("empty segment")
		}

		if strings.Contains(segment, "/") {
			parts := strings.Split(segment, "/")
			if len(parts) != 2 {
				return cronField{}, fmt.Errorf("invalid step syntax: %q", segment)
			}
			step, err := strconv.Atoi(parts[1])
			if err != nil || step <= 0 {
				return cronField{}, fmt.Errorf("invalid step value in %q", segment)
			}

			start := min
			end := max
			base := parts[0]
			if base != "*" {
				rangeStart, rangeEnd, rangeErr := parseRange(base, min, max)
				if rangeErr != nil {
					return cronField{}, rangeErr
				}
				start = rangeStart
				end = rangeEnd
			}

			for i := start; i <= end; i += step {
				allowed[i] = true
			}
			continue
		}

		if strings.Contains(segment, "-") {
			rangeStart, rangeEnd, err := parseRange(segment, min, max)
			if err != nil {
				return cronField{}, err
			}
			for i := rangeStart; i <= rangeEnd; i++ {
				allowed[i] = true
			}
			continue
		}

		value, err := strconv.Atoi(segment)
		if err != nil {
			return cronField{}, fmt.Errorf("invalid value %q", segment)
		}
		if value < min || value > max {
			return cronField{}, fmt.Errorf("value %d out of range [%d,%d]", value, min, max)
		}
		allowed[value] = true
	}

	return cronField{allowed: allowed}, nil
}

func parseRange(raw string, min int, max int) (int, int, error) {
	parts := strings.Split(raw, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid range syntax: %q", raw)
	}

	start, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid range start in %q", raw)
	}
	end, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid range end in %q", raw)
	}

	if start > end {
		return 0, 0, fmt.Errorf("range start > end in %q", raw)
	}
	if start < min || end > max {
		return 0, 0, fmt.Errorf("range %d-%d out of bounds [%d,%d]", start, end, min, max)
	}

	return start, end, nil
}
