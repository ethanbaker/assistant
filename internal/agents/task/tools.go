package task

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nlpodyssey/openai-agents-go/agents"
	"github.com/openai/openai-go/v2/packages/param"
)

// registerTools registers all task management tools
func (ta *TaskAgent) registerTools() {
	ta.agent.WithTools(
		ta.createFetchTasksTool(),
		ta.createGetTaskDetailsTool(),
		ta.createGetUpcomingTasksTool(),
		ta.createGetRecurringTasksTool(),
		ta.createNewTaskTool(),
		ta.createUpdateTaskTool(),
		ta.createCompleteTaskTool(),
		ta.createHighlightBlockersTool(),
		ta.createSuggestFocusAreasToolF(),
	)
}

/** ---- TOOL ARGUMENT STRUCTURES ---- **/

type FetchTasksArgs struct {
	Complete *bool   `json:"complete,omitempty"`
	Priority *string `json:"priority,omitempty"`
	Effort   *string `json:"effort,omitempty"`
	DueDate  *string `json:"due_date,omitempty"` // ISO format date
	Project  *string `json:"project,omitempty"`
}

type GetTaskDetailsArgs struct {
	TaskID string `json:"task_id"`
}

type CreateTaskArgs struct {
	Title    string  `json:"title"`
	Priority *string `json:"priority,omitempty"`
	Effort   *string `json:"effort,omitempty"`
	DueDate  *string `json:"due_date,omitempty"` // ISO format date
	Project  *string `json:"project,omitempty"`
}

type UpdateTaskArgs struct {
	TaskID   string  `json:"task_id"`
	Title    *string `json:"title,omitempty"`
	Priority *string `json:"priority,omitempty"`
	Effort   *string `json:"effort,omitempty"`
	DueDate  *string `json:"due_date,omitempty"` // ISO format date
	Project  *string `json:"project,omitempty"`
}

type CompleteTaskArgs struct {
	TaskID string `json:"task_id"`
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
					"type":        "boolean",
					"description": "Filter by completion status (optional)",
				},
				"priority": map[string]any{
					"type":        "string",
					"description": "Filter by priority level (optional)",
					"enum":        []string{PRIORITY_LOW, PRIORITY_MEDIUM, PRIORITY_HIGH, PRIORITY_CRITICAL},
				},
				"effort": map[string]any{
					"type":        "string",
					"description": "Filter by effort level (optional)",
					"enum":        []string{EFFORT_LOW, EFFORT_MEDIUM, EFFORT_HIGH},
				},
				"due_date": map[string]any{
					"type":        "string",
					"description": "Filter by due date in YYYY-MM-DD format (optional)",
				},
				"project": map[string]any{
					"type":        "string",
					"description": "Filter by project label (optional)",
				},
			},
			"additionalProperties": false,
			"required":             []string{"complete", "priority", "effort", "due_date", "project"},
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return ta.handleFetchTasks(ctx, arguments)
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
					"type":        "string",
					"description": "Priority level for the task (optional)",
					"enum":        []string{PRIORITY_LOW, PRIORITY_MEDIUM, PRIORITY_HIGH, PRIORITY_CRITICAL},
				},
				"effort": map[string]any{
					"type":        "string",
					"description": "Effort level for the task (optional)",
					"enum":        []string{EFFORT_LOW, EFFORT_MEDIUM, EFFORT_HIGH},
				},
				"due_date": map[string]any{
					"type":        "string",
					"description": "Due date in YYYY-MM-DD format (optional)",
				},
				"project": map[string]any{
					"type":        "string",
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
					"type":        "string",
					"description": "The title of the task",
				},
				"priority": map[string]any{
					"type":        "string",
					"description": "Priority level for the task (optional)",
					"enum":        []string{PRIORITY_LOW, PRIORITY_MEDIUM, PRIORITY_HIGH, PRIORITY_CRITICAL},
				},
				"effort": map[string]any{
					"type":        "string",
					"description": "Effort level for the task (optional)",
					"enum":        []string{EFFORT_LOW, EFFORT_MEDIUM, EFFORT_HIGH},
				},
				"due_date": map[string]any{
					"type":        "string",
					"description": "Due date in YYYY-MM-DD format (optional)",
				},
				"project": map[string]any{
					"type":        "string",
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

// createSuggestFocusAreasToolF creates the suggest focus areas tool
func (ta *TaskAgent) createSuggestFocusAreasToolF() agents.FunctionTool {
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
	var args FetchTasksArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if !isValidPriority(args.Priority) {
		return nil, fmt.Errorf("invalid priority value: %s", *args.Priority)
	}

	if !isValidEffort(args.Effort) {
		return nil, fmt.Errorf("invalid effort value: %s", *args.Effort)
	}

	if !isValidDate(args.DueDate) {
		return nil, fmt.Errorf("invalid due_date format: %s", *args.DueDate)
	}

	query := ta.buildFetchTasksQuery(args)
	return ta.queryTasks(ctx, query)
}

// handleGetTaskDetails processes the get task details tool invocation
func (ta *TaskAgent) handleGetTaskDetails(ctx context.Context, arguments string) (any, error) {
	var args GetTaskDetailsArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if args.TaskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	return ta.getTaskDetails(ctx, args.TaskID)
}

// handleGetUpcomingTasks processes the get upcoming tasks tool invocation
func (ta *TaskAgent) handleGetUpcomingTasks(ctx context.Context, arguments string) (any, error) {
	query := ta.buildUpcomingTasksQuery()
	return ta.queryTasks(ctx, query)
}

// handleGetRecurringTasks processes the get recurring tasks tool invocation
func (ta *TaskAgent) handleGetRecurringTasks(ctx context.Context, arguments string) (any, error) {
	query := ta.buildRecurringTasksQuery()
	return ta.queryRecurringTasks(ctx, query)
}

// handleCreateNewTask processes the create new task tool invocation
func (ta *TaskAgent) handleCreateNewTask(ctx context.Context, arguments string) (any, error) {
	var args CreateTaskArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if args.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	if !isValidPriority(args.Priority) {
		return nil, fmt.Errorf("invalid priority value: %s", *args.Priority)
	}

	if !isValidEffort(args.Effort) {
		return nil, fmt.Errorf("invalid effort value: %s", *args.Effort)
	}

	if !isValidDate(args.DueDate) {
		return nil, fmt.Errorf("invalid due_date format: %s", *args.DueDate)
	}

	return ta.createTask(ctx, args)
}

// handleUpdateTask processes the update task tool invocation
func (ta *TaskAgent) handleUpdateTask(ctx context.Context, arguments string) (any, error) {
	var args UpdateTaskArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if args.TaskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	if args.Title == nil || *args.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	if !isValidPriority(args.Priority) {
		return nil, fmt.Errorf("invalid priority value: %s", *args.Priority)
	}

	if !isValidEffort(args.Effort) {
		return nil, fmt.Errorf("invalid effort value: %s", *args.Effort)
	}

	if !isValidDate(args.DueDate) {
		return nil, fmt.Errorf("invalid due_date format: %s", *args.DueDate)
	}

	return ta.updateTask(ctx, args)
}

// handleCompleteTask processes the complete task tool invocation
func (ta *TaskAgent) handleCompleteTask(ctx context.Context, arguments string) (any, error) {
	var args CompleteTaskArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if args.TaskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	return ta.completeTask(ctx, args.TaskID)
}

// handleHighlightBlockers processes the highlight blockers tool invocation
func (ta *TaskAgent) handleHighlightBlockers(ctx context.Context, arguments string) (any, error) {
	return ta.highlightBlockers(ctx)
}

// handleSuggestFocusAreas processes the suggest focus areas tool invocation
func (ta *TaskAgent) handleSuggestFocusAreas(ctx context.Context, arguments string) (any, error) {
	return ta.suggestFocusAreas(ctx)
}
