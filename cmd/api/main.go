package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/ethanbaker/api/pkg/utils"
	communication_agent "github.com/ethanbaker/assistant/internal/agents/communication"
	entry_agent "github.com/ethanbaker/assistant/internal/agents/entry"
	overseer_agent "github.com/ethanbaker/assistant/internal/agents/overseer"
	planner_agent "github.com/ethanbaker/assistant/internal/agents/planner"
	schedule_agent "github.com/ethanbaker/assistant/internal/agents/schedule"
	task_agent "github.com/ethanbaker/assistant/internal/agents/task"
	agent_api "github.com/ethanbaker/assistant/internal/api/modules/agent"
	health_api "github.com/ethanbaker/assistant/internal/api/modules/health"
	outreach_api "github.com/ethanbaker/assistant/internal/api/modules/outreach"
	"github.com/ethanbaker/assistant/internal/api/routes"
	"github.com/ethanbaker/assistant/internal/database"
	"github.com/ethanbaker/assistant/internal/domain"
	dailydigest_outreach "github.com/ethanbaker/assistant/internal/outreaches/daily_digest"
	notionschedule_outreach "github.com/ethanbaker/assistant/internal/outreaches/notion_schedule"
	"github.com/ethanbaker/assistant/internal/outreaches/test"
	client_repo "github.com/ethanbaker/assistant/internal/repositories/client"
	job_repo "github.com/ethanbaker/assistant/internal/repositories/job"
	jobexecution_repo "github.com/ethanbaker/assistant/internal/repositories/jobexecution"
	jobsubscription_repo "github.com/ethanbaker/assistant/internal/repositories/jobsubscription"
	fact_repo "github.com/ethanbaker/assistant/internal/repositories/keyfact"
	session_repo "github.com/ethanbaker/assistant/internal/repositories/session"
	"github.com/ethanbaker/assistant/internal/services/gcal"
	"github.com/ethanbaker/assistant/internal/services/memory"
	"github.com/ethanbaker/assistant/internal/services/notion"
	"github.com/ethanbaker/assistant/internal/services/searxng"
	"github.com/ethanbaker/assistant/pkg/config"
	"github.com/gin-gonic/gin"
	"github.com/goccy/go-yaml"
)

// Start the API server
func main() {
	// 1.1. Load config
	envFile := ".env"
	if val := os.Getenv("ENV_FILE"); val != "" {
		envFile = val
	}
	config.Load(envFile)

	// 1.2. Load config files
	calendarConfig, err := loadCalendarServiceConfig()
	fatalOnErr(err)

	// 1.3. Reference app-wide config properities
	promptBasePath := config.MustGetenv("PROMPT_FILES_BASE_PATH")

	// 2. Database connections
	db, err := database.NewMySQLConnection(&database.MySQLConfig{
		User:            config.MustGetenv("MYSQL_USERNAME"),
		Password:        config.MustGetenv("MYSQL_ROOT_PASSWORD"),
		Host:            config.MustGetenv("MYSQL_HOST"),
		Port:            config.MustGetenv("MYSQL_PORT"),
		Database:        config.MustGetenv("MYSQL_DATABASE"),
		Loc:             config.MustGetenv("MYSQL_LOC"),
		Charset:         config.GetenvWithDefault("MYSQL_CHARSET", "utf8mb4"),
		LogLevel:        config.GetenvWithDefault("MYSQL_LOG_LEVEL", "INFO"),
		ParseTime:       true,
		MaxIdleConns:    1,
		MaxOpenConns:    10,
		ConnMaxLifetime: time.Hour,
	})
	fatalOnErr(err)
	defer database.Close(db)

	// 3. Repositories
	factRepo, err := fact_repo.NewMySQLRepository(db)
	fatalOnErr(err)
	sessionRepo, err := session_repo.NewMySQLRepository(db)
	fatalOnErr(err)
	jobRepo, err := job_repo.NewMySQLRepository(db)
	fatalOnErr(err)
	clientRepo, err := client_repo.NewMySQLRepository(db)
	fatalOnErr(err)
	jobSubscriptionRepo, err := jobsubscription_repo.NewMySQLRepository(db)
	fatalOnErr(err)
	jobExecutionRepo, err := jobexecution_repo.NewMySQLRepository(db)
	fatalOnErr(err)

	// 4.1. Agent Services
	gcalService, err := gcal.NewCalendarService(calendarConfig)
	fatalOnErr(err)
	notionTaskService, err := notion.NewNotionTaskService(notion.NotionTaskServiceConfig{
		APIToken:            config.GetenvWithDefault("NOTION_API_TOKEN", ""),
		TasksDatabaseID:     config.GetenvWithDefault("NOTION_DATABASE_TASKS_ID", ""),
		RecurringDatabaseID: config.GetenvWithDefault("NOTION_DATABASE_RECURRING_ID", ""),
		ScheduleDatabaseID:  config.GetenvWithDefault("NOTION_DATABASE_SCHEDULE_ID", ""),
		Timezone:            config.MustGetenv("TIMEZONE"),
	})
	fatalOnErr(err)

	// 4.2. Domain-level Agents
	communicationAgent, err := communication_agent.NewCommunicationAgent(communication_agent.CommunicationAgentConfig{
		Model:               "gpt-4o-mini",
		PromptFile:          path.Join(promptBasePath, "communication-agent.md"),
		TelegramAppID:       config.GetenvWithDefault("TG_APP_ID", ""),
		TelegramAPIHash:     config.GetenvWithDefault("TG_API_HASH", ""),
		TelegramSessionFile: config.GetenvWithDefault("TG_SESSION_PATH", ""),
	})
	fatalOnErr(err)
	scheduleAgent, err := schedule_agent.NewScheduleAgent(schedule_agent.ScheduleAgentConfig{
		Model:           "gpt-4o-mini",
		PromptFile:      path.Join(promptBasePath, "schedule-agent.md"),
		CalendarService: gcalService,
	})
	fatalOnErr(err)
	taskAgent, err := task_agent.NewTaskAgent(task_agent.TaskAgentConfig{
		Model:             "gpt-4o-mini",
		PromptFile:        path.Join(promptBasePath, "task-agent.md"),
		NotionTaskService: notionTaskService,
	})
	fatalOnErr(err)

	domainAgents := []domain.Handoff{
		{
			ToolName:        "DIRECT_communication_agent",
			ToolDescription: "Delegate to the agent specializing in communication",
			Agent:           communicationAgent,
		},
		{
			ToolName:        "DIRECT_schedule_agent",
			ToolDescription: "Delegate to the agent specializing in calendar-related events",
			Agent:           scheduleAgent,
		},
		{
			ToolName:        "DIRECT_task_agent",
			ToolDescription: "Delegate to the agent specalizing in task/todo lists",
			Agent:           taskAgent,
		},
	}

	// 4.3. Agent Tool Registries
	memoryRegistry, err := memory.NewMemoryToolRegistry(memory.MemoryToolRegistryConfig{
		FactRepository:    factRepo,
		SessionRepository: sessionRepo,
	})
	fatalOnErr(err)
	memoryRegistry.RegisterTools(communicationAgent, scheduleAgent, taskAgent)

	searxngRegistry, err := searxng.NewSearxngToolRegister(searxng.SearxngToolRegisterConfig{
		SearxngUrl: config.GetenvWithDefault("SEARXNG_URL", ""),
	})
	fatalOnErr(err)
	searxngRegistry.RegisterTools(communicationAgent, scheduleAgent, taskAgent)

	// 4.4 High-level Agents
	overseerAgent, err := overseer_agent.NewOverseerAgent(overseer_agent.OverseerAgentConfig{
		Model:      "gpt-4o-mini",
		PromptFile: path.Join(promptBasePath, "overseer-agent.md"),
		Handoffs:   domainAgents,
	})
	fatalOnErr(err)
	plannerAgent, err := planner_agent.NewPlannerAgent(planner_agent.PlannerAgentConfig{
		Model:      "gpt-4o-mini",
		PromptFile: path.Join(promptBasePath, "planner-agent.md"),
		Overseer:   overseerAgent,
	})
	fatalOnErr(err)

	entryHandoffs := []domain.Handoff{
		{
			ToolName:        "PLANNER",
			ToolDescription: "Delegate to the planner agent to plan complex, multi-agent tasks",
			Agent:           plannerAgent,
		},
	}
	entryHandoffs = append(entryHandoffs, domainAgents...)
	entryAgent, err := entry_agent.NewEntryAgent(entry_agent.EntryAgentConfig{
		Model:      "gpt-4o-mini",
		PromptFile: path.Join(promptBasePath, "entry-agent.md"),
		Handoffs:   entryHandoffs,
	})
	fatalOnErr(err)

	// 5. Outreaches
	dailyDigest := dailydigest_outreach.NewDailyDigest(gcalService, notionTaskService)
	notionSchedule := notionschedule_outreach.NewNotionSchedule(context.Background(), notionTaskService)
	outreaches := map[string]outreach_api.HandlerFunc{
		"example-job":     test.TestOutreach,
		"daily-digest":    dailyDigest.RunDailyDigest,
		"notion-schedule": notionSchedule.RunNotionSchedule,
	}

	// 5.1. Api Services
	agentService := agent_api.NewService(agent_api.ServiceConfig{
		SessionRepository:   sessionRepo,
		ExecutionRepository: jobExecutionRepo,
		EntryAgent:          entryAgent,
	})
	outreachClientService := outreach_api.NewClientService(clientRepo)
	outreachJobService := outreach_api.NewJobService(jobRepo)
	outreachSubscriptionService := outreach_api.NewSubscriptionService(clientRepo, jobRepo, jobSubscriptionRepo)
	outreachExecutor := outreach_api.NewExecutor(outreach_api.ExecutorConfig{
		JobRepository:          jobRepo,
		ClientRepository:       clientRepo,
		SubscriptionRepository: jobSubscriptionRepo,
		ExecutionRepository:    jobExecutionRepo,
		Handlers:               outreaches,
	})
	fatalOnErr(outreachExecutor.Start(context.Background()))
	defer outreachExecutor.Stop()

	// 5.2. Api Handlers
	agentHandler := agent_api.NewHandler(agent_api.HandlerConfig{
		Service:          agentService,
		ClientRepository: clientRepo,
	})
	healthHandler := health_api.NewHandler()
	outreachHandler := outreach_api.NewHandler(outreach_api.HandlerConfig{
		AdminKey:            config.MustGetenv("OUTREACH_ADMIN_KEY"),
		JobService:          outreachJobService,
		ClientService:       outreachClientService,
		SubscriptionService: outreachSubscriptionService,
	})

	// 7. Router
	engine := gin.Default()
	engine.NoRoute(utils.NoRouteHandler)
	engine.SetTrustedProxies(nil)

	public := engine.Group("")
	internal := engine.Group("")
	//internal.Use ... use middleware

	// 8. Routes
	public.GET(routes.GET_HEALTH, healthHandler.GetStatus)

	internal.POST(routes.POST_CREATE_SESSION, agentHandler.CreateSession)
	internal.GET(routes.GET_SESSION_BY_UUID, agentHandler.GetSession)
	internal.POST(routes.POST_MESSAGE_TO_SESSION, agentHandler.PostMessage)
	internal.DELETE(routes.DELETE_SESSION, agentHandler.DeleteSession)
	internal.POST(routes.POST_SESSION_JOB_EXECUTION_CONTEXT, agentHandler.AttachJobExecutionContext)

	internal.POST(routes.POST_OUTREACH_JOB, outreachHandler.CreateJob)
	internal.POST(routes.POST_OUTREACH_CLIENT, outreachHandler.RegisterClient)
	internal.POST(routes.POST_OUTREACH_SUBSCRIPTION, outreachHandler.Subscribe)
	internal.DELETE(routes.DELETE_OUTREACH_SUBSCRIPTION, outreachHandler.Unsubscribe)

	// 9. Start
	port := config.GetenvWithDefault("PORT", "8080")
	if err := engine.Run(":" + port); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}

// Helper method to fatally log errors
func fatalOnErr(err error) {
	if err != nil {
		log.Fatalf("Application start failed: %v\n", err)
	}
}

// Helper method to load calendar service config
func loadCalendarServiceConfig() (gcal.CalendarServiceConfig, error) {
	calendarConfigPath := config.GetenvWithDefault("GOOGLE_CALENDAR_CONFIG_FILE", "resources/config/google/config.yml")

	yamlFile, err := os.ReadFile(calendarConfigPath)
	if err != nil {
		return gcal.CalendarServiceConfig{}, fmt.Errorf("failed to read calendar config file: %w", err)
	}

	// calendars represents the structure of the calendar configuration file
	var cfg gcal.CalendarServiceConfig
	if err := yaml.Unmarshal(yamlFile, &cfg); err != nil {
		return gcal.CalendarServiceConfig{}, fmt.Errorf("failed to parse calendar config yaml: %w", err)
	}

	return cfg, nil
}
