package notion

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	notionapi "github.com/dstotijn/go-notion"
)

// NotionTaskServiceConfig holds configuration for the NotionTaskService
type NotionTaskServiceConfig struct {
	APIToken            string
	TasksDatabaseID     string
	RecurringDatabaseID string
}

// NotionTaskService provides Notion API access for task management
type NotionTaskService struct {
	client              *notionapi.Client
	tasksDatabaseID     string
	recurringDatabaseID string
}

// NewNotionTaskService creates a new NotionTaskService
func NewNotionTaskService(cfg NotionTaskServiceConfig) (*NotionTaskService, error) {
	if cfg.APIToken == "" {
		return nil, errors.New("APIToken is required")
	}
	if cfg.TasksDatabaseID == "" {
		return nil, errors.New("TasksDatabaseID is required")
	}
	if cfg.RecurringDatabaseID == "" {
		return nil, errors.New("RecurringDatabaseID is required")
	}

	client := notionapi.NewClient(cfg.APIToken, notionapi.WithHTTPClient(&http.Client{
		Timeout: 20 * time.Second,
	}))

	return &NotionTaskService{
		client:              client,
		tasksDatabaseID:     cfg.TasksDatabaseID,
		recurringDatabaseID: cfg.RecurringDatabaseID,
	}, nil
}

// QueryTasks executes a Notion database query and returns tasks
func (ns *NotionTaskService) QueryTasks(ctx context.Context, args FetchTasksArgs) ([]Task, error) {
	if ns.tasksDatabaseID == "" {
		return nil, fmt.Errorf("tasks database ID not configured")
	}

	response, err := ns.client.QueryDatabase(ctx, ns.tasksDatabaseID, ns.buildFetchTasksQuery(args))
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}

	return ns.pagesToTasks(response.Results), nil
}

// QueryUpcomingTasks queries upcoming tasks in the database
func (ns *NotionTaskService) QueryUpcomingTasks(ctx context.Context) ([]Task, error) {
	if ns.tasksDatabaseID == "" {
		return nil, fmt.Errorf("tasks database ID not configured")
	}

	response, err := ns.client.QueryDatabase(ctx, ns.tasksDatabaseID, ns.buildUpcomingTasksQuery())
	if err != nil {
		return nil, fmt.Errorf("failed to query upcoming tasks: %w", err)
	}

	return ns.pagesToTasks(response.Results), nil
}

// QueryRecurringTasks queries the recurring tasks database with provided filters
func (ns *NotionTaskService) QueryRecurringTasks(ctx context.Context) ([]Task, error) {
	if ns.recurringDatabaseID == "" {
		return nil, fmt.Errorf("recurring tasks database ID not configured")
	}

	response, err := ns.client.QueryDatabase(ctx, ns.recurringDatabaseID, ns.buildRecurringTasksQuery())
	if err != nil {
		return nil, fmt.Errorf("failed to query recurring tasks: %w", err)
	}

	return ns.pagesToTasks(response.Results), nil
}

// GetTaskDetails retrieves detailed information about a specific task
func (ns *NotionTaskService) GetTaskDetails(ctx context.Context, taskID string) (TaskDetails, error) {
	page, err := ns.client.FindPageByID(ctx, taskID)
	if err != nil {
		return TaskDetails{}, fmt.Errorf("failed to get task details: %w", err)
	}

	blocks, err := ns.client.FindBlockChildrenByID(ctx, taskID, nil)
	if err != nil {
		return TaskDetails{}, fmt.Errorf("failed to get task content: %w", err)
	}

	return TaskDetails{
		Task:    ns.pageToTask(&page),
		Content: ns.formatBlocks(blocks.Results),
	}, nil
}

// CreateTask creates a new task in the Notion database
func (ns *NotionTaskService) CreateTask(ctx context.Context, args CreateTaskArgs) (Task, error) {
	if ns.tasksDatabaseID == "" {
		return Task{}, fmt.Errorf("tasks database ID not configured")
	}

	// Prepare properties for the new task
	properties := notionapi.DatabasePageProperties{
		COLUMN_TITLE: notionapi.DatabasePageProperty{
			Title: []notionapi.RichText{
				{
					Type: notionapi.RichTextTypeText,
					Text: &notionapi.Text{Content: args.Title},
				},
			},
		},
		COLUMN_COMPLETE: notionapi.DatabasePageProperty{
			Checkbox: pointer(false),
		},
	}

	// Add optional properties if provided
	if args.Priority != nil {
		properties[COLUMN_PRIORITY] = notionapi.DatabasePageProperty{
			Select: &notionapi.SelectOptions{Name: *args.Priority},
		}
	}

	if args.Effort != nil {
		properties[COLUMN_EFFORT] = notionapi.DatabasePageProperty{
			Select: &notionapi.SelectOptions{Name: *args.Effort},
		}
	}

	if args.DueDate != nil {
		dueDate, err := time.Parse(DATE_FORMAT, *args.DueDate)
		if err == nil {
			properties[COLUMN_DATE] = notionapi.DatabasePageProperty{
				Date: &notionapi.Date{Start: notionapi.NewDateTime(dueDate, false)},
			}
		}
	}

	if args.Project != nil {
		/* TODO: figure out relations
		properties["Project"] = notionapi.DatabasePageProperty{
			Select: &notionapi.SelectOptions{Name: *args.Project},
		}
		*/
	}

	// Create the new task page
	page, err := ns.client.CreatePage(ctx, notionapi.CreatePageParams{
		ParentType:             notionapi.ParentTypeDatabase,
		ParentID:               ns.tasksDatabaseID,
		DatabasePageProperties: &properties,
	})
	if err != nil {
		return Task{}, fmt.Errorf("failed to create task: %w", err)
	}

	return ns.pageToTask(&page), nil
}

// UpdateTask updates properties of an existing task
func (ns *NotionTaskService) UpdateTask(ctx context.Context, args UpdateTaskArgs) (Task, error) {
	properties := notionapi.DatabasePageProperties{}

	// Update provided properties
	if args.Title != nil {
		properties[COLUMN_TITLE] = notionapi.DatabasePageProperty{
			Title: []notionapi.RichText{
				{
					Type: notionapi.RichTextTypeText,
					Text: &notionapi.Text{Content: *args.Title},
				},
			},
		}
	}

	if args.Priority != nil {
		properties[COLUMN_PRIORITY] = notionapi.DatabasePageProperty{
			Select: &notionapi.SelectOptions{Name: *args.Priority},
		}
	}

	if args.Effort != nil {
		properties[COLUMN_EFFORT] = notionapi.DatabasePageProperty{
			Select: &notionapi.SelectOptions{Name: *args.Effort},
		}
	}

	if args.DueDate != nil {
		dueDate, err := time.Parse(DATE_FORMAT, *args.DueDate)
		if err == nil {
			properties[COLUMN_DATE] = notionapi.DatabasePageProperty{
				Date: &notionapi.Date{Start: notionapi.NewDateTime(dueDate, false)},
			}
		}
	}

	/* TODO: figure out relations
	if args.Project != nil {
		properties[COLUMN_PROJECT] = notionapi.DatabasePageProperty{
			Select: &notionapi.SelectOptions{Name: *args.Project},
		}
	}
	*/

	page, err := ns.client.UpdatePage(ctx, args.TaskID, notionapi.UpdatePageParams{
		DatabasePageProperties: properties,
	})
	if err != nil {
		return Task{}, fmt.Errorf("failed to update task: %w", err)
	}

	return ns.pageToTask(&page), nil
}

// CompleteTask marks a task as complete
func (ns *NotionTaskService) CompleteTask(ctx context.Context, taskID string) (Task, error) {
	properties := notionapi.DatabasePageProperties{
		COLUMN_COMPLETE: notionapi.DatabasePageProperty{
			Checkbox: pointer(true),
		},
	}

	page, err := ns.client.UpdatePage(ctx, taskID, notionapi.UpdatePageParams{
		DatabasePageProperties: properties,
	})
	if err != nil {
		return Task{}, fmt.Errorf("failed to complete task: %w", err)
	}

	return ns.pageToTask(&page), nil
}

// HighlightBlockers returns overdue and critical priority tasks
func (ns *NotionTaskService) HighlightBlockers(ctx context.Context) (HighlightBlockersResult, error) {
	if ns.tasksDatabaseID == "" {
		return HighlightBlockersResult{}, fmt.Errorf("tasks database ID not configured")
	}

	overdueTasksResponse, err := ns.client.QueryDatabase(ctx, ns.tasksDatabaseID, ns.buildOverdueTasksQuery())
	if err != nil {
		return HighlightBlockersResult{}, fmt.Errorf("failed to get overdue tasks: %w", err)
	}
	overdueTasks := ns.pagesToTasks(overdueTasksResponse.Results)

	criticalTasksResponse, err := ns.client.QueryDatabase(ctx, ns.tasksDatabaseID, ns.buildCriticalTasksQuery())
	if err != nil {
		return HighlightBlockersResult{}, fmt.Errorf("failed to get critical tasks: %w", err)
	}
	criticalTasks := ns.pagesToTasks(criticalTasksResponse.Results)

	return HighlightBlockersResult{
		OverdueTasks:  overdueTasks,
		CriticalTasks: criticalTasks,
	}, nil
}

// SuggestFocusAreas returns urgent high-priority tasks and quick wins
func (ns *NotionTaskService) SuggestFocusAreas(ctx context.Context) (SuggestFocusAreasResult, error) {
	urgentResponse, err := ns.client.QueryDatabase(ctx, ns.tasksDatabaseID, ns.buildUrgentTasksQuery())
	if err != nil {
		return SuggestFocusAreasResult{}, fmt.Errorf("failed to get urgent tasks: %w", err)
	}
	urgentTasks := ns.pagesToTasks(urgentResponse.Results)

	quickWinsResponse, err := ns.client.QueryDatabase(ctx, ns.tasksDatabaseID, ns.buildQuickWinsQuery())
	if err != nil {
		return SuggestFocusAreasResult{}, fmt.Errorf("failed to get quick win tasks: %w", err)
	}
	quickWins := ns.pagesToTasks(quickWinsResponse.Results)

	return SuggestFocusAreasResult{
		UrgentTasks: urgentTasks,
		QuickWins:   quickWins,
	}, nil
}
