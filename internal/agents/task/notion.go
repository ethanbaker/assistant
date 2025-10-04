package task

import (
	"context"
	"fmt"
	"time"

	notionapi "github.com/dstotijn/go-notion"
)

// Implementation methods

// queryTasks executes a Notion database query and returns formatted results
func (ta *TaskAgent) queryTasks(ctx context.Context, query notionapi.DatabaseQuery) (any, error) {
	// Handle dry run mode
	if ta.ShouldDryRun(ctx) {
		return map[string]any{
			"message": "DRY RUN: Would query tasks with provided filters",
			"query":   query,
		}, nil
	}

	// Get database ID from config
	databaseID := ta.config.Get("NOTION_DATABASE_TASKS_ID")
	if databaseID == "" {
		return nil, fmt.Errorf("NOTION_DATABASE_TASKS_ID not configured")
	}

	// Execute the query
	client := ta.getNotionClient()
	response, err := client.QueryDatabase(ctx, databaseID, &query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}

	// Format the response and return
	return ta.formatTasksResponse(response.Results), nil
}

// queryRecurringTasks queries the recurring tasks database with provided filters
func (ta *TaskAgent) queryRecurringTasks(ctx context.Context, query notionapi.DatabaseQuery) (any, error) {
	if ta.ShouldDryRun(ctx) {
		return map[string]any{
			"message": "DRY RUN: Would query recurring tasks",
			"query":   query,
		}, nil
	}

	databaseID := ta.config.Get("NOTION_DATABASE_RECURRING_ID")
	if databaseID == "" {
		return nil, fmt.Errorf("NOTION_DATABASE_RECURRING_ID not configured")
	}

	client := ta.getNotionClient()
	response, err := client.QueryDatabase(ctx, databaseID, &query)
	if err != nil {
		return nil, fmt.Errorf("failed to query recurring tasks: %w", err)
	}

	return ta.formatTasksResponse(response.Results), nil
}

// getTaskDetails retrieves detailed information about a specific task
func (ta *TaskAgent) getTaskDetails(ctx context.Context, taskID string) (any, error) {
	if ta.ShouldDryRun(ctx) {
		return map[string]any{
			"message": "DRY RUN: Would get task details",
			"task_id": taskID,
		}, nil
	}

	// Fetch the task page
	client := ta.getNotionClient()
	page, err := client.FindPageByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task details: %w", err)
	}

	// Get page content
	blocks, err := client.FindBlockChildrenByID(ctx, taskID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get task content: %w", err)
	}

	return map[string]any{
		"task":    ta.formatTask(&page),
		"content": ta.formatBlocks(blocks.Results),
	}, nil
}

// createTask creates a new task in the Notion database
func (ta *TaskAgent) createTask(ctx context.Context, args CreateTaskArgs) (any, error) {
	// Handle dry run mode
	if ta.ShouldDryRun(ctx) {
		return map[string]any{
			"message": "DRY RUN: Would create new task",
			"args":    args,
		}, nil
	}

	// Get database ID from config
	databaseID := ta.config.Get("NOTION_DATABASE_TASKS_ID")
	if databaseID == "" {
		return nil, fmt.Errorf("NOTION_DATABASE_TASKS_ID not configured")
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
	client := ta.getNotionClient()
	page, err := client.CreatePage(ctx, notionapi.CreatePageParams{
		ParentType:             notionapi.ParentTypeDatabase,
		ParentID:               databaseID,
		DatabasePageProperties: &properties,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// Return success response
	return map[string]any{
		"message": "Task created successfully",
		"task":    ta.formatTask(&page),
	}, nil
}

// updateTask updates properties of an existing task
func (ta *TaskAgent) updateTask(ctx context.Context, args UpdateTaskArgs) (any, error) {
	// Handle dry run mode
	if ta.ShouldDryRun(ctx) {
		return map[string]any{
			"message": "DRY RUN: Would update task",
			"args":    args,
		}, nil
	}

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

	client := ta.getNotionClient()
	page, err := client.UpdatePage(ctx, args.TaskID, notionapi.UpdatePageParams{
		DatabasePageProperties: properties,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	return map[string]any{
		"message": "Task updated successfully",
		"task":    ta.formatTask(&page),
	}, nil
}

// completeTask marks a task as complete
func (ta *TaskAgent) completeTask(ctx context.Context, taskID string) (any, error) {
	// Handle dry run mode
	if ta.ShouldDryRun(ctx) {
		return map[string]any{
			"message": "DRY RUN: Would complete task",
			"task_id": taskID,
		}, nil
	}

	// Set the Complete property to true
	properties := notionapi.DatabasePageProperties{
		COLUMN_COMPLETE: notionapi.DatabasePageProperty{
			Checkbox: pointer(true),
		},
	}

	// Update the task page with the new complete status
	client := ta.getNotionClient()
	page, err := client.UpdatePage(ctx, taskID, notionapi.UpdatePageParams{
		DatabasePageProperties: properties,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to complete task: %w", err)
	}

	// Return success response
	return map[string]any{
		"message": "Task completed successfully",
		"task":    ta.formatTask(&page),
	}, nil
}

func (ta *TaskAgent) highlightBlockers(ctx context.Context) (any, error) {
	// Get overdue tasks
	overdueQuery := ta.buildOverdueTasksQuery()
	overdueTasks, err := ta.queryTasks(ctx, overdueQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get overdue tasks: %w", err)
	}

	// Get critical priority tasks
	criticalQuery := ta.buildCriticalTasksQuery()
	criticalTasks, err := ta.queryTasks(ctx, criticalQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get critical tasks: %w", err)
	}

	// Return the identified blockers
	return map[string]any{
		"overdue_tasks":  overdueTasks,
		"critical_tasks": criticalTasks,
		"message":        "Identified potential blockers: overdue and critical priority tasks",
	}, nil
}

func (ta *TaskAgent) suggestFocusAreas(ctx context.Context) (any, error) {
	// Get high priority tasks due soon
	urgentQuery := ta.buildUrgentTasksQuery()
	urgentTasks, err := ta.queryTasks(ctx, urgentQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get urgent tasks: %w", err)
	}

	// Get small effort tasks for quick wins
	quickWinsQuery := ta.buildQuickWinsQuery()
	quickWins, err := ta.queryTasks(ctx, quickWinsQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get quick win tasks: %w", err)
	}

	return map[string]any{
		"urgent_tasks": urgentTasks,
		"quick_wins":   quickWins,
		"message":      "Focus areas: urgent high-priority tasks and quick wins for momentum",
	}, nil
}
