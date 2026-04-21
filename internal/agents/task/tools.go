package task

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	notion_service "github.com/ethanbaker/assistant/internal/services/notion"
	"github.com/nlpodyssey/openai-agents-go/agents"
	"github.com/openai/openai-go/v2/packages/param"
)

type GetTaskDetailsArgs struct {
	TaskID string `json:"task_id"`
}

type CompleteTaskArgs struct {
	TaskID string `json:"task_id"`
}

// registerTools registers all task management tools
func (ta *TaskAgent) registerTools() {
	ta.agent.WithTools(
		ta.createFetchTasksTool(),
		ta.createGetTodaysTaskTool(),
		ta.createGetTaskDetailsTool(),
		ta.createGetUpcomingTasksTool(),
		ta.createGetRecurringTasksTool(),
		ta.createNewTaskTool(),
		ta.createUpdateTaskTool(),
		ta.createCompleteTaskTool(),
		ta.createHighlightBlockersTool(),
		ta.createSuggestFocusAreasTool(),
	)
}

/** ---- TOOL CREATORS ---- **/

// createFetchTasksTool creates the fetch tasks tool
func (ta *TaskAgent) createFetchTasksTool() agents.FunctionTool {
	return agents.FunctionTool{
		Name:        "fetch_tasks",
		Description: "Retrieve tasks by filters (complete checkbox, priority, effort, due date, project label)",
		ParamsJSONSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"complete": map[string]any{
					"type":        []string{"boolean", "null"},
					"description": "Filter by completion status (optional)",
				},
				"priority": map[string]any{
					"type":        []string{"string", "null"},
					"description": "Filter by priority level (optional)",
					"enum":        []any{notion_service.PRIORITY_LOW, notion_service.PRIORITY_MEDIUM, notion_service.PRIORITY_HIGH, notion_service.PRIORITY_CRITICAL, nil},
				},
				"effort": map[string]any{
					"type":        []string{"string", "null"},
					"description": "Filter by effort level (optional)",
					"enum":        []any{notion_service.EFFORT_LOW, notion_service.EFFORT_MEDIUM, notion_service.EFFORT_HIGH, nil},
				},
				"due_date": map[string]any{
					"type":        []string{"string", "null"},
					"description": "Filter by due date in YYYY-MM-DD format (optional)",
				},
				"project": map[string]any{
					"type":        []string{"string", "null"},
					"description": "Filter by project label (optional)",
				},
			},
			"required":             []string{"complete", "priority", "effort", "due_date", "project"},
			"additionalProperties": false,
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return ta.handleFetchTasks(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}
}

// createGetTodaysTaskTool creates the get today's tasks tool
func (ta *TaskAgent) createGetTodaysTaskTool() agents.FunctionTool {
	return agents.FunctionTool{
		Name:        "get_todays_tasks",
		Description: "Get all tasks that are due today",
		ParamsJSONSchema: map[string]any{
			"type":                 "object",
			"properties":           map[string]any{},
			"additionalProperties": false,
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return ta.handleGetTodaysTasks(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}
}

// createGetTaskDetailsTool creates the get task details tool
func (ta *TaskAgent) createGetTaskDetailsTool() agents.FunctionTool {
	return agents.FunctionTool{
		Name:        "get_task_details",
		Description: "Retrieve detailed information about a specific task by its Notion ID",
		ParamsJSONSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"task_id": map[string]any{
					"type":        "string",
					"description": "The Notion page ID of the task",
				},
			},
			"additionalProperties": false,
			"required":             []string{"task_id"},
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return ta.handleGetTaskDetails(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}
}

// createGetUpcomingTasksTool creates the get upcoming tasks tool
func (ta *TaskAgent) createGetUpcomingTasksTool() agents.FunctionTool {
	return agents.FunctionTool{
		Name:        "get_upcoming_tasks",
		Description: "Get upcoming tasks. This includes tasks that are due soon or have high priority",
		ParamsJSONSchema: map[string]any{
			"type":                 "object",
			"properties":           map[string]any{},
			"additionalProperties": false,
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return ta.handleGetUpcomingTasks(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}
}

// createGetRecurringTasksTool creates the get recurring tasks tool
func (ta *TaskAgent) createGetRecurringTasksTool() agents.FunctionTool {
	return agents.FunctionTool{
		Name:        "get_recurring_tasks",
		Description: "Get all recurring tasks that are due today",
		ParamsJSONSchema: map[string]any{
			"type":                 "object",
			"properties":           map[string]any{},
			"additionalProperties": false,
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return ta.handleGetRecurringTasks(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}
}

// createNewTaskTool creates the new task tool
func (ta *TaskAgent) createNewTaskTool() agents.FunctionTool {
	return agents.FunctionTool{
		Name:        "create_new_task",
		Description: "Add a new task to the tasks database",
		ParamsJSONSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title": map[string]any{
					"type":        "string",
					"description": "The title of the new task",
				},
				"priority": map[string]any{
					"type":        []string{"string", "null"},
					"description": "Priority level for the task (optional)",
					"enum":        []any{notion_service.PRIORITY_LOW, notion_service.PRIORITY_MEDIUM, notion_service.PRIORITY_HIGH, notion_service.PRIORITY_CRITICAL, nil},
				},
				"effort": map[string]any{
					"type":        []string{"string", "null"},
					"description": "Effort level for the task (optional)",
					"enum":        []any{notion_service.EFFORT_LOW, notion_service.EFFORT_MEDIUM, notion_service.EFFORT_HIGH, nil},
				},
				"due_date": map[string]any{
					"type":        []string{"string", "null"},
					"description": "Due date in YYYY-MM-DD format (optional)",
				},
				"project": map[string]any{
					"type":        []string{"string", "null"},
					"description": "Project label for the task (optional)",
				},
			},
			"additionalProperties": false,
			"required":             []string{"title", "priority", "effort", "due_date", "project"},
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return ta.handleCreateNewTask(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}
}

// createUpdateTaskTool creates the update task tool
func (ta *TaskAgent) createUpdateTaskTool() agents.FunctionTool {
	return agents.FunctionTool{
		Name:        "update_task",
		Description: "Update a task's properties (effort, priority, due date, project)",
		ParamsJSONSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"task_id": map[string]any{
					"type":        "string",
					"description": "The Notion page ID of the task to update",
				},
				"title": map[string]any{
					"type":        []string{"string", "null"},
					"description": "The title of the task (optional)",
				},
				"priority": map[string]any{
					"type":        []string{"string", "null"},
					"description": "Priority level for the task (optional)",
					"enum":        []any{notion_service.PRIORITY_LOW, notion_service.PRIORITY_MEDIUM, notion_service.PRIORITY_HIGH, notion_service.PRIORITY_CRITICAL, nil},
				},
				"effort": map[string]any{
					"type":        []string{"string", "null"},
					"description": "Effort level for the task (optional)",
					"enum":        []any{notion_service.EFFORT_LOW, notion_service.EFFORT_MEDIUM, notion_service.EFFORT_HIGH, nil},
				},
				"due_date": map[string]any{
					"type":        []string{"string", "null"},
					"description": "Due date in YYYY-MM-DD format (optional)",
				},
				"project": map[string]any{
					"type":        []string{"string", "null"},
					"description": "Project label for the task (optional)",
				},
			},
			"additionalProperties": false,
			"required":             []string{"task_id", "title", "priority", "effort", "due_date", "project"},
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return ta.handleUpdateTask(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}
}

// createCompleteTaskTool creates the complete task tool
func (ta *TaskAgent) createCompleteTaskTool() agents.FunctionTool {
	return agents.FunctionTool{
		Name:        "complete_task",
		Description: "Mark a task as complete by setting the complete checkbox to true",
		ParamsJSONSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"task_id": map[string]any{
					"type":        "string",
					"description": "The Notion page ID of the task to complete",
				},
			},
			"additionalProperties": false,
			"required":             []string{"task_id"},
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return ta.handleCompleteTask(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}
}

// createHighlightBlockersTool creates the highlight blockers tool
func (ta *TaskAgent) createHighlightBlockersTool() agents.FunctionTool {
	return agents.FunctionTool{
		Name:        "highlight_blockers",
		Description: "Identify overdue or high-priority tasks that may be blocking progress",
		ParamsJSONSchema: map[string]any{
			"type":                 "object",
			"properties":           map[string]any{},
			"additionalProperties": false,
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return ta.handleHighlightBlockers(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}
}

// createSuggestFocusAreasTool creates the suggest focus areas tool
func (ta *TaskAgent) createSuggestFocusAreasTool() agents.FunctionTool {
	return agents.FunctionTool{
		Name:        "suggest_focus_areas",
		Description: "Recommend what to work on next based on deadlines, priority, and effort",
		ParamsJSONSchema: map[string]any{
			"type":                 "object",
			"properties":           map[string]any{},
			"additionalProperties": false,
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return ta.handleSuggestFocusAreas(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}
}

/** ---- TOOL HANDLERS ---- **/

// handleFetchTasks processes the fetch tasks tool invocation
func (ta *TaskAgent) handleFetchTasks(ctx context.Context, arguments string) (any, error) {
	arguments = normalizeArguments(arguments)

	var args notion_service.FetchTasksArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if !notion_service.IsValidPriority(args.Priority) {
		return nil, fmt.Errorf("invalid priority value: %s", valueOrEmpty(args.Priority))
	}

	if !notion_service.IsValidEffort(args.Effort) {
		return nil, fmt.Errorf("invalid effort value: %s", valueOrEmpty(args.Effort))
	}

	if !notion_service.IsValidDate(args.DueDate) {
		return nil, fmt.Errorf("invalid due_date format: %s", valueOrEmpty(args.DueDate))
	}

	tasks, err := ta.taskService.QueryTasks(ctx, args)
	if err != nil {
		return nil, err
	}
	return formatTasksResponse(tasks), nil
}

// handleGetTodaysTasks processes the get today's tasks tool invocation
func (ta *TaskAgent) handleGetTodaysTasks(ctx context.Context, arguments string) (any, error) {
	upcomingTasks, err := ta.taskService.QueryUpcomingTasks(ctx)
	if err != nil {
		return nil, err
	}

	recurringTasks, err := ta.taskService.QueryRecurringTasks(ctx)
	if err != nil {
		return nil, err
	}

	return formatTasksAndRecurringTasksResponse(upcomingTasks, recurringTasks), nil
}

// handleGetTaskDetails processes the get task details tool invocation
func (ta *TaskAgent) handleGetTaskDetails(ctx context.Context, arguments string) (any, error) {
	arguments = normalizeArguments(arguments)

	var args GetTaskDetailsArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if args.TaskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	details, err := ta.taskService.GetTaskDetails(ctx, args.TaskID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"task": details.Task, "content": details.Content}, nil
}

// handleGetUpcomingTasks processes the get upcoming tasks tool invocation
func (ta *TaskAgent) handleGetUpcomingTasks(ctx context.Context, arguments string) (any, error) {
	tasks, err := ta.taskService.QueryUpcomingTasks(ctx)
	if err != nil {
		return nil, err
	}
	return formatTasksResponse(tasks), nil
}

// handleGetRecurringTasks processes the get recurring tasks tool invocation
func (ta *TaskAgent) handleGetRecurringTasks(ctx context.Context, arguments string) (any, error) {
	tasks, err := ta.taskService.QueryRecurringTasks(ctx)
	if err != nil {
		return nil, err
	}
	return formatRecurringTasksResponse(tasks), nil
}

// handleCreateNewTask processes the create new task tool invocation
func (ta *TaskAgent) handleCreateNewTask(ctx context.Context, arguments string) (any, error) {
	arguments = normalizeArguments(arguments)

	var args notion_service.CreateTaskArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if args.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	if !notion_service.IsValidPriority(args.Priority) {
		return nil, fmt.Errorf("invalid priority value: %s", valueOrEmpty(args.Priority))
	}

	if !notion_service.IsValidEffort(args.Effort) {
		return nil, fmt.Errorf("invalid effort value: %s", valueOrEmpty(args.Effort))
	}

	if !notion_service.IsValidDate(args.DueDate) {
		return nil, fmt.Errorf("invalid due_date format: %s", valueOrEmpty(args.DueDate))
	}

	task, err := ta.taskService.CreateTask(ctx, args)
	if err != nil {
		return nil, err
	}
	return map[string]any{"message": "Task created successfully", "task": task}, nil
}

// handleUpdateTask processes the update task tool invocation
func (ta *TaskAgent) handleUpdateTask(ctx context.Context, arguments string) (any, error) {
	arguments = normalizeArguments(arguments)

	var args notion_service.UpdateTaskArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if args.TaskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	if !notion_service.IsValidPriority(args.Priority) {
		return nil, fmt.Errorf("invalid priority value: %s", valueOrEmpty(args.Priority))
	}

	if !notion_service.IsValidEffort(args.Effort) {
		return nil, fmt.Errorf("invalid effort value: %s", valueOrEmpty(args.Effort))
	}

	if !notion_service.IsValidDate(args.DueDate) {
		return nil, fmt.Errorf("invalid due_date format: %s", valueOrEmpty(args.DueDate))
	}

	task, err := ta.taskService.UpdateTask(ctx, args)
	if err != nil {
		return nil, err
	}
	return map[string]any{"message": "Task updated successfully", "task": task}, nil
}

// handleCompleteTask processes the complete task tool invocation
func (ta *TaskAgent) handleCompleteTask(ctx context.Context, arguments string) (any, error) {
	arguments = normalizeArguments(arguments)

	var args CompleteTaskArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if args.TaskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	task, err := ta.taskService.CompleteTask(ctx, args.TaskID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"message": "Task completed successfully", "task": task}, nil
}

// handleHighlightBlockers processes the highlight blockers tool invocation
func (ta *TaskAgent) handleHighlightBlockers(ctx context.Context, arguments string) (any, error) {
	result, err := ta.taskService.HighlightBlockers(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"overdue_tasks":  result.OverdueTasks,
		"critical_tasks": result.CriticalTasks,
		"message":        "Identified potential blockers: overdue and critical priority tasks",
	}, nil
}

// handleSuggestFocusAreas processes the suggest focus areas tool invocation
func (ta *TaskAgent) handleSuggestFocusAreas(ctx context.Context, arguments string) (any, error) {
	result, err := ta.taskService.SuggestFocusAreas(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"urgent_tasks": result.UrgentTasks,
		"quick_wins":   result.QuickWins,
		"message":      "Focus areas: urgent high-priority tasks and quick wins for momentum",
	}, nil
}

func normalizeArguments(arguments string) string {
	if strings.TrimSpace(arguments) == "" {
		return "{}"
	}

	return arguments
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

// formatTasksResponse formats a slice of tasks for tool response; adds message when empty
func formatTasksResponse(tasks []notion_service.Task) map[string]any {
	out := map[string]any{"tasks": tasks}
	if len(tasks) == 0 {
		out["message"] = "No tasks found"
	}
	return out
}

// formatRecurringTasksResponse formats a slice of recurring tasks for tool response; adds message when empty
func formatRecurringTasksResponse(tasks []notion_service.RecurringTask) map[string]any {
	out := map[string]any{"recurring_tasks": tasks}
	if len(tasks) == 0 {
		out["message"] = "No tasks found"
	}
	return out
}

// formatTasksAndRecurringTasksResponse formats a slice of tasks and recurring tasks for tool response
func formatTasksAndRecurringTasksResponse(tasks []notion_service.Task, recurringTasks []notion_service.RecurringTask) map[string]any {
	out := map[string]any{"recurring_tasks": tasks, "tasks": recurringTasks}
	if len(tasks) == 0 && len(recurringTasks) == 0 {
		out["message"] = "No tasks found"
	}
	return out
}
