package notion

import (
	"fmt"
	"strings"
	"time"

	notionapi "github.com/dstotijn/go-notion"
)

// pointer is a helper function for building a pointer to a primitive type
//
//go:fix inline
func pointer[T any](v T) *T {
	return new(v)
}

// pagesToTasks converts Notion API page results into Task slices
func (ns *NotionTaskService) pagesToTasks(pages []notionapi.Page) []Task {
	if len(pages) == 0 {
		return nil
	}

	tasks := make([]Task, 0, len(pages))
	for i := range pages {
		tasks = append(tasks, ns.pageToTask(&pages[i]))
	}
	return tasks
}

// pageToTask converts a single Notion page into a Task
func (ns *NotionTaskService) pageToTask(page *notionapi.Page) Task {
	task := Task{ID: page.ID}
	if page.Properties == nil {
		return task
	}

	props, ok := page.Properties.(notionapi.DatabasePageProperties)
	if !ok {
		return task
	}

	if name, ok := props[COLUMN_TITLE]; ok && name.Title != nil && len(name.Title) > 0 {
		task.Title = name.Title[0].Text.Content
	}
	if complete, ok := props[COLUMN_COMPLETE]; ok && complete.Checkbox != nil {
		task.Complete = *complete.Checkbox
	}
	if priority, ok := props[COLUMN_PRIORITY]; ok && priority.Select != nil {
		task.Priority = &priority.Select.Name
	}
	if effort, ok := props[COLUMN_EFFORT]; ok && effort.Select != nil {
		task.Effort = &effort.Select.Name
	}
	if date, ok := props[COLUMN_DATE]; ok && date.Date != nil {
		d := date.Date.Start.Format(DATE_FORMAT)
		p := date.Date.Start.Format(PRETTY_DATE_FORMAT)
		task.DueDate = &d
		task.DueDatePretty = &p
	}
	if project, ok := props[COLUMN_PROJECT]; ok && project.Formula != nil {
		task.Project = project.Formula.String
	}
	return task
}

// pagesToRecurringTasks converts Notion API page results into RecurringTask slices
func (ns *NotionTaskService) pagesToRecurringTasks(pages []notionapi.Page) []RecurringTask {
	if len(pages) == 0 {
		return nil
	}

	tasks := make([]RecurringTask, 0, len(pages))
	for i := range pages {
		tasks = append(tasks, ns.pageToRecurringTask(&pages[i]))
	}
	return tasks
}

// pageToRecurringTask converts a single Notion page into a RecurringTask
func (ns *NotionTaskService) pageToRecurringTask(page *notionapi.Page) RecurringTask {
	task := RecurringTask{ID: page.ID}
	if page.Properties == nil {
		return task
	}

	props, ok := page.Properties.(notionapi.DatabasePageProperties)
	if !ok {
		return task
	}

	if name, ok := props[RECURRING_COLUMN_TITLE]; ok && name.Title != nil && len(name.Title) > 0 {
		task.Title = name.Title[0].Text.Content
	}
	if complete, ok := props[RECURRING_COLUMN_DONE]; ok && complete.Checkbox != nil {
		task.Done = *complete.Checkbox
	}
	if date, ok := props[RECURRING_COLUMN_LAST_DONE]; ok && date.Date != nil {
		d := date.Date.Start.Format(DATE_FORMAT)
		p := date.Date.Start.Format(PRETTY_DATE_FORMAT)
		task.LastDone = &d
		task.LastDonePretty = &p
	}
	if date, ok := props[RECURRING_COLUMN_DATE]; ok && date.Formula != nil {
		d := date.Formula.Date.Start.Format(DATE_FORMAT)
		p := date.Formula.Date.Start.Format(PRETTY_DATE_FORMAT)
		task.DueDate = &d
		task.DueDatePretty = &p
	}
	if t, ok := props[RECURRING_COLUMN_TYPE]; ok && t.Select != nil {
		task.Type = &t.Select.Name
	}
	if connection, ok := props[RECURRING_COLUMN_CONNECTION]; ok && connection.Formula != nil {
		task.Connection = connection.Formula.String
	}
	return task
}

// pagesToScheduleItems converts Notion API page results into ScheduleItem slices
func (ns *NotionTaskService) pagesToScheduleItems(pages []notionapi.Page) []ScheduleItem {
	if len(pages) == 0 {
		return nil
	}

	items := make([]ScheduleItem, 0, len(pages))
	for i := range pages {
		items = append(items, ns.pageToScheduleItem(&pages[i]))
	}
	return items
}

// pageToScheduleItem converts a single Notion page into a Task
func (ns *NotionTaskService) pageToScheduleItem(page *notionapi.Page) ScheduleItem {
	item := ScheduleItem{ID: page.ID}
	if page.Properties == nil {
		return item
	}

	props, ok := page.Properties.(notionapi.DatabasePageProperties)
	if !ok {
		return item
	}

	if name, ok := props[SCHEDULE_COLUMN_TITLE]; ok && name.Title != nil && len(name.Title) > 0 {
		item.Title = name.Title[0].Text.Content
	}
	if date, ok := props[SCHEDULE_COLUMN_DATE]; ok && date.Date != nil {
		item.Start = &date.Date.Start.Time

		if date.Date.End != nil {
			item.End = &date.Date.End.Time
		}
	}
	if timespan, ok := props[SCHEDULE_COLUMN_TIMESPAN]; ok && timespan.Formula != nil {
		// Timespan is broken -> notion calculates formula at API call in UTC instead of EST >:/
		//item.Timespan = timespan.Formula.String

		if item.End != nil && item.Start.Format(DATE_FORMAT) == item.End.Format(DATE_FORMAT) {
			s := item.Start.Format(PRETTY_TIME_FORMAT)
			e := item.End.Format(PRETTY_TIME_FORMAT)
			item.Timespan = new(fmt.Sprintf("%s → %s", s, e))
		} else {
			item.Timespan = new("All Day")
		}
	}
	if project, ok := props[SCHEDULE_COLUMN_PROJECT]; ok && project.Formula != nil {
		item.Project = project.Formula.String
	}
	return item
}

// formatBlocks formats Notion blocks into simplified structures
func (ns *NotionTaskService) formatBlocks(blocks []notionapi.Block) []map[string]any {
	if len(blocks) == 0 {
		return []map[string]any{
			{"message": "No content found"},
		}
	}

	var content []map[string]any
	for _, block := range blocks {
		content = append(content, ns.formatBlock(&block))
	}
	return content
}

// formatBlock formats a single Notion block into a simplified structure
func (ns *NotionTaskService) formatBlock(block *notionapi.Block) map[string]any {
	// TODO: implement
	return map[string]any{}
}

// IsValidPriority validates a priority value
func IsValidPriority(priority *string) bool {
	if priority == nil {
		return true
	} else if strings.TrimSpace(*priority) == "" {
		return true
	}

	validPriorities := map[string]bool{
		PRIORITY_NONE:   true,
		PRIORITY_LOW:    true,
		PRIORITY_MEDIUM: true,
		PRIORITY_HIGH:   true,
	}
	return validPriorities[*priority]
}

// IsValidEffort validates an effort value
func IsValidEffort(effort *string) bool {
	if effort == nil {
		return true
	} else if strings.TrimSpace(*effort) == "" {
		return true
	}

	validEfforts := map[string]bool{
		EFFORT_LOW:    true,
		EFFORT_MEDIUM: true,
		EFFORT_HIGH:   true,
	}
	return validEfforts[*effort]
}

// IsValidDate validates a date value
func IsValidDate(date *string) bool {
	if date == nil {
		return true
	} else if strings.TrimSpace(*date) == "" {
		return true
	}

	_, err := time.Parse(DATE_FORMAT, *date)
	if err != nil {
		_, err = time.Parse(PRETTY_DATE_FORMAT, *date)
	}
	return err == nil
}
