package task

import (
	"strings"
	"time"

	notionapi "github.com/dstotijn/go-notion"
)

// Helper function for building a pointer to a primitive type
func pointer[T any](v T) *T {
	return &v
}

// Format Notion API responses into simplified structures
func (ta *TaskAgent) formatTasksResponse(pages []notionapi.Page) []map[string]any {
	// If no pages, return message
	if len(pages) == 0 {
		return []map[string]any{
			{"message": "No tasks found"},
		}
	}

	// Format each page into a simplified task structure
	var tasks []map[string]any
	for _, page := range pages {
		tasks = append(tasks, ta.formatTask(&page))
	}
	return tasks
}

// formatTask is a helper function that formats a single Notion page into a simplified task structure
func (ta *TaskAgent) formatTask(page *notionapi.Page) map[string]any {
	// Basic task info
	task := map[string]any{
		"id": page.ID,
	}

	// Extract properties
	if page.Properties != nil {
		props, ok := page.Properties.(notionapi.DatabasePageProperties)
		if !ok {
			return task
		}

		if name, ok := props[COLUMN_TITLE]; ok && name.Title != nil {
			if len(name.Title) > 0 {
				task["title"] = name.Title[0].Text.Content
			}
		}

		if complete, ok := props[COLUMN_COMPLETE]; ok && complete.Checkbox != nil {
			task["complete"] = *complete.Checkbox
		}

		if priority, ok := props[COLUMN_PRIORITY]; ok && priority.Select != nil {
			task["priority"] = priority.Select.Name
		}

		if effort, ok := props[COLUMN_EFFORT]; ok && effort.Select != nil {
			task["effort"] = effort.Select.Name
		}

		if date, ok := props[COLUMN_DATE]; ok && date.Date != nil {
			task["due_date"] = date.Date.Start.Format(DATE_FORMAT)
			task["due_date_pretty"] = date.Date.Start.Format(PRETTY_DATE_FORMAT)
		}

		if project, ok := props[COLUMN_PROJECT]; ok && project.Select != nil {
			task["project"] = project.Select.Name
		}
	}

	return task
}

// formatBlocks is a helper function that formats Notion blocks into simplified structures
func (ta *TaskAgent) formatBlocks(blocks []notionapi.Block) []map[string]any {
	// If no blocks, return empty message
	if len(blocks) == 0 {
		return []map[string]any{
			{"message": "No content found"},
		}
	}

	// Format each block
	var content []map[string]any
	for _, block := range blocks {
		content = append(content, ta.formatBlock(&block))
	}
	return content
}

// formatBlock is a helper function that formats a single Notion block into a simplified structure
func (ta *TaskAgent) formatBlock(block *notionapi.Block) map[string]any {
	// TODO: implement
	return map[string]any{}
}

// Validate priority values
func isValidPriority(priority *string) bool {
	if priority == nil {
		return true // Allow nil (no priority set)
	} else if strings.TrimSpace(*priority) == "" {
		return true // Allow empty string (no priority set)
	}

	validPriorities := map[string]bool{
		PRIORITY_NONE:   true,
		PRIORITY_LOW:    true,
		PRIORITY_MEDIUM: true,
		PRIORITY_HIGH:   true,
	}
	return validPriorities[*priority]
}

// Validate effort values
func isValidEffort(effort *string) bool {
	if effort == nil {
		return true // Allow nil (no effort set)
	} else if strings.TrimSpace(*effort) == "" {
		return true // Allow empty string (no effort set)
	}

	validEfforts := map[string]bool{
		EFFORT_LOW:    true,
		EFFORT_MEDIUM: true,
		EFFORT_HIGH:   true,
	}
	return validEfforts[*effort]
}

// Validate date values
func isValidDate(date *string) bool {
	if date == nil {
		return true // Allow nil (no date set)
	} else if strings.TrimSpace(*date) == "" {
		return true // Allow empty string (no date set)
	}

	_, err := time.Parse(DATE_FORMAT, *date)
	if err != nil {
		_, err = time.Parse(PRETTY_DATE_FORMAT, *date)
	}
	return err == nil
}
