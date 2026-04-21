package notion

import (
	"context"
	"time"
)

// ScheduleItem represents a schedule item from the Notion service
type ScheduleItem struct {
	ID       string     `json:"id"`
	Title    string     `json:"title"`
	Start    *time.Time `json:"start,omitempty"`
	End      *time.Time `json:"end,omitempty"`
	Timespan *string    `json:"timespan,omitempty"`
	Project  *string    `json:"project,omitempty"`
}

// RecurringTask represents a recurring task from the Notion task service
type RecurringTask struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	Done           bool    `json:"done"`
	LastDone       *string `json:"last_done,omitempty"`
	LastDonePretty *string `json:"last_done_pretty,omitempty"`
	DueDate        *string `json:"due_date,omitempty"`
	DueDatePretty  *string `json:"due_date_pretty,omitempty"`
	Type           *string `json:"type,omitempty"`
	Connection     *string `json:"connection,omitempty"`
}

// Task represents a task from the Notion task service
type Task struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	Complete      bool    `json:"complete"`
	Priority      *string `json:"priority,omitempty"`
	Effort        *string `json:"effort,omitempty"`
	DueDate       *string `json:"due_date,omitempty"`        // DATE_FORMAT
	DueDatePretty *string `json:"due_date_pretty,omitempty"` // PRETTY_DATE_FORMAT
	Project       *string `json:"project,omitempty"`
}

// TaskDetails holds a task and its page content (blocks)
type TaskDetails struct {
	Task    Task             `json:"task"`
	Content []map[string]any `json:"content"`
}

// HighlightBlockersResult holds overdue and critical tasks
type HighlightBlockersResult struct {
	OverdueTasks  []Task `json:"overdue_tasks"`
	CriticalTasks []Task `json:"critical_tasks"`
}

// SuggestFocusAreasResult holds urgent and quick-win tasks
type SuggestFocusAreasResult struct {
	UrgentTasks []Task `json:"urgent_tasks"`
	QuickWins   []Task `json:"quick_wins"`
}

// FetchTasksArgs holds arguments for fetching tasks with filters
type FetchTasksArgs struct {
	Complete *bool   `json:"complete,omitempty"`
	Priority *string `json:"priority,omitempty"`
	Effort   *string `json:"effort,omitempty"`
	DueDate  *string `json:"due_date,omitempty"` // ISO format date
	Project  *string `json:"project,omitempty"`
}

// CreateTaskArgs holds arguments for creating a new task
type CreateTaskArgs struct {
	Title    string  `json:"title"`
	Priority *string `json:"priority,omitempty"`
	Effort   *string `json:"effort,omitempty"`
	DueDate  *string `json:"due_date,omitempty"` // ISO format date
	Project  *string `json:"project,omitempty"`
}

// UpdateTaskArgs holds arguments for updating an existing task
type UpdateTaskArgs struct {
	TaskID   string  `json:"task_id"`
	Title    *string `json:"title,omitempty"`
	Priority *string `json:"priority,omitempty"`
	Effort   *string `json:"effort,omitempty"`
	DueDate  *string `json:"due_date,omitempty"` // ISO format date
	Project  *string `json:"project,omitempty"`
}

// TaskService defines the interface for task management operations
type TaskService interface {
	QueryTasks(ctx context.Context, args FetchTasksArgs) ([]Task, error)
	QueryUpcomingTasks(ctx context.Context) ([]Task, error)
	QueryRecurringTasks(ctx context.Context) ([]RecurringTask, error)
	QueryScheduleItems(ctx context.Context) ([]ScheduleItem, error)
	GetTaskDetails(ctx context.Context, taskID string) (TaskDetails, error)
	CreateTask(ctx context.Context, args CreateTaskArgs) (Task, error)
	UpdateTask(ctx context.Context, args UpdateTaskArgs) (Task, error)
	CompleteTask(ctx context.Context, taskID string) (Task, error)
	HighlightBlockers(ctx context.Context) (HighlightBlockersResult, error)
	SuggestFocusAreas(ctx context.Context) (SuggestFocusAreasResult, error)
}

// Ensure NotionTaskService implements TaskService at compile time
var _ TaskService = (*NotionTaskService)(nil)
