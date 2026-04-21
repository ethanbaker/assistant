package dailydigest

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ethanbaker/assistant/internal/api/modules/outreach"
	"github.com/ethanbaker/assistant/internal/services/gcal"
	"github.com/ethanbaker/assistant/internal/services/notion"
)

const timeFormat = "3:04 PM"

// DailyDigest is the outreach implementation for the daily digest
type DailyDigest struct {
	calendarService gcal.CalendarService
	notionService   notion.TaskService
}

// NewDailyDigest creates a new DailyDigest outreach
func NewDailyDigest(calendarService gcal.CalendarService, notionService notion.TaskService) *DailyDigest {
	return &DailyDigest{
		calendarService: calendarService,
		notionService:   notionService,
	}
}

// RunDailyDigest generates the daily digest output
func (d *DailyDigest) RunDailyDigest(ctx context.Context, params json.RawMessage) (*outreach.OutreachResponse, error) {
	var output string

	calendarSection, err := d.getCalendarSection(ctx)
	if err != nil {
		return nil, err
	}
	output += calendarSection

	upcomingSection, err := d.getUpcomingTasksSection(ctx)
	if err != nil {
		return nil, err
	}
	output += upcomingSection

	criticalSection, err := d.getCriticalTasksSection(ctx)
	if err != nil {
		return nil, err
	}
	output += criticalSection

	recurringSection, err := d.getRecurringTasksSection(ctx)
	if err != nil {
		return nil, err
	}
	output += recurringSection

	return &outreach.OutreachResponse{
		Content: output,
	}, nil
}

// calendarEntry is an internal type used to sort and format calendar events
type calendarEntry struct {
	startTime time.Time
	timespan  string
	title     string
}

func (d *DailyDigest) getCalendarSection(ctx context.Context) (string, error) {
	events, err := d.calendarService.GetTodayEvents(ctx, "")
	if err != nil {
		return "", fmt.Errorf("failed to get calendar events: %w", err)
	}

	tz := d.calendarService.GetTimezone()
	var entries []calendarEntry

	// Add calendar events
	for _, event := range events {
		if event.Summary == "" {
			continue
		}

		// Skip "Busy" events (time-blocking duplicates)
		if strings.EqualFold(strings.TrimSpace(event.Summary), "busy") {
			continue
		}

		var startTime time.Time
		var timespan string

		if event.Start != nil && event.Start.DateTime != "" {
			t, err := time.Parse(time.RFC3339, event.Start.DateTime)
			if err != nil {
				continue
			}
			startTime = t.In(tz)

			if event.End != nil && event.End.DateTime != "" {
				endTime, err := time.Parse(time.RFC3339, event.End.DateTime)
				if err == nil {
					timespan = fmt.Sprintf("%s → %s", startTime.Format(timeFormat), endTime.In(tz).Format(timeFormat))
				} else {
					timespan = startTime.Format(timeFormat)
				}
			} else {
				timespan = startTime.Format(timeFormat)
			}
		} else if event.Start != nil && event.Start.Date != "" {
			// All-day event: use midnight so it sorts before timed events
			t, err := time.Parse("2006-01-02", event.Start.Date)
			if err != nil {
				continue
			}
			startTime = t.In(tz)
			timespan = "All Day"
		} else {
			continue
		}

		entries = append(entries, calendarEntry{
			startTime: startTime,
			timespan:  timespan,
			title:     event.Summary,
		})
	}

	// Add notion schedule events
	scheduleEvents, err := d.notionService.QueryScheduleItems(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get schedule events: %w", err)
	}

	for _, event := range scheduleEvents {
		if event.Title == "" {
			continue
		}

		var startTime time.Time
		if event.Start != nil {
			startTime = (*event.Start).In(tz)
		}

		if event.Timespan == nil || *event.Timespan == "" {
			continue
		}
		timespan := *event.Timespan

		entries = append(entries, calendarEntry{
			startTime: startTime,
			timespan:  timespan,
			title:     event.Title,
		})
	}

	// Sort entries by time
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].startTime.Before(entries[j].startTime)
	})

	if len(entries) == 0 {
		return "", nil
	}

	var output strings.Builder
	output.WriteString("<STRONG>Schedule:</STRONG>\n")
	for _, entry := range entries {
		fmt.Fprintf(&output, "- %s: %s\n", entry.timespan, entry.title)
	}

	return output.String(), nil
}

func (d *DailyDigest) getUpcomingTasksSection(ctx context.Context) (string, error) {
	tasks, err := d.notionService.QueryUpcomingTasks(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get upcoming tasks: %w", err)
	}

	if len(tasks) == 0 {
		return "", nil
	}

	var output strings.Builder
	output.WriteString("\n<STRONG>Upcoming Tasks:</STRONG>\n")
	for _, task := range tasks {
		project := ""
		if task.Project != nil && *task.Project != "" {
			project = fmt.Sprintf("<EM>%s</EM>", *task.Project)
		}

		date := ""
		if task.DueDatePretty != nil && *task.DueDatePretty != "" {
			date = fmt.Sprintf("(%s)", *task.DueDatePretty)
		}

		fmt.Fprintf(&output, "- %s %s %s\n", task.Title, project, date)
	}

	return output.String(), nil
}

func (d *DailyDigest) getCriticalTasksSection(ctx context.Context) (string, error) {
	notComplete := false
	criticalPriority := notion.PRIORITY_CRITICAL
	tasks, err := d.notionService.QueryTasks(ctx, notion.FetchTasksArgs{
		Complete: &notComplete,
		Priority: &criticalPriority,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get critical tasks: %w", err)
	}

	if len(tasks) == 0 {
		return "", nil
	}

	var output strings.Builder
	output.WriteString("\n<STRONG>Critical Tasks:</STRONG>\n")
	for _, task := range tasks {
		project := ""
		if task.Project != nil && *task.Project != "" {
			project = fmt.Sprintf("(<EM>%s</EM>)", *task.Project)
		}

		date := ""
		if task.DueDatePretty != nil && *task.DueDatePretty != "" {
			date = fmt.Sprintf("(%s)", *task.DueDatePretty)
		}

		fmt.Fprintf(&output, "- %s %s %s\n", task.Title, project, date)
	}

	return output.String(), nil
}

func (d *DailyDigest) getRecurringTasksSection(ctx context.Context) (string, error) {
	tasks, err := d.notionService.QueryRecurringTasks(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get recurring tasks: %w", err)
	}

	if len(tasks) == 0 {
		return "", nil
	}

	var output strings.Builder
	output.WriteString("\n<STRONG>Recurring Tasks:</STRONG>\n")
	for _, task := range tasks {
		if task.Type != nil && strings.ToLower(*task.Type) == "connection" && task.Connection != nil {
			fmt.Fprintf(&output, "- %s with %s\n", task.Title, *task.Connection)
		} else {
			fmt.Fprintf(&output, "- %s\n", task.Title)
		}
	}

	return output.String(), nil
}
