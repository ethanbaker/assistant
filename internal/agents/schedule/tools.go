package schedule

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	gcal_service "github.com/ethanbaker/assistant/internal/services/gcal"
	"github.com/ethanbaker/assistant/pkg/logger"
	"github.com/nlpodyssey/openai-agents-go/agents"
	"github.com/openai/openai-go/v2/packages/param"
)

// registerTools registers all schedule management tools
func (sa *ScheduleAgent) registerTools() {
	sa.agent.WithTools(
		sa.createSearchEventsTools(),
		sa.createGetTodayEventsTools(),
		sa.createGetWeekEventsTools(),
		sa.createGetSpecificDayEventsTools(),
		sa.createCreateEventTools(),
		sa.createUpdateEventTools(),
		sa.createDeleteEventTools(),
	)
}

/** ---- TOOL ARGUMENT STRUCTURES ---- **/

type SearchEventsArgs struct {
	Query        string  `json:"query"`
	CalendarName *string `json:"calendar_name,omitempty"`
}

type GetEventsArgs struct {
	CalendarName *string `json:"calendar_name,omitempty"`
}

type GetSpecificDayEventsArgs struct {
	Date         string  `json:"date"` // YYYY-MM-DD format
	CalendarName *string `json:"calendar_name,omitempty"`
}

type CreateEventArgs struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	StartTime    string `json:"start_time"` // RFC3339 format
	EndTime      string `json:"end_time"`   // RFC3339 format
	CalendarName string `json:"calendar_name"`
}

type UpdateEventArgs struct {
	EventID      string  `json:"event_id"`
	CalendarName string  `json:"calendar_name"`
	Title        *string `json:"title,omitempty"`
	Description  *string `json:"description,omitempty"`
	StartTime    *string `json:"start_time,omitempty"` // RFC3339 format
	EndTime      *string `json:"end_time,omitempty"`   // RFC3339 format
}

type DeleteEventArgs struct {
	EventID      string `json:"event_id"`
	CalendarName string `json:"calendar_name"`
}

/** ---- TOOL CREATORS ---- **/

// createSearchEventsTools creates the search events tool
func (sa *ScheduleAgent) createSearchEventsTools() agents.FunctionTool {
	calendarNames := sa.calendarService.GetCalendarNamesList()
	calendarNamesWithNull := enumWithNull(calendarNames)

	return agents.FunctionTool{
		Name:        "search_calendar_events",
		Description: "Search for calendar events by name/query and optionally filter by calendar name",
		ParamsJSONSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "Search query to find events by name or description",
				},
				"calendar_name": map[string]any{
					"type":        []string{"string", "null"},
					"description": "Calendar name to search in (optional - if omitted, searches all calendars)",
					"enum":        calendarNamesWithNull,
				},
			},
			"additionalProperties": false,
			"required":             []string{"query", "calendar_name"},
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return sa.handleSearchEvents(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}
}

// createGetTodayEventsTools creates the get today's events tool
func (sa *ScheduleAgent) createGetTodayEventsTools() agents.FunctionTool {
	calendarNames := sa.calendarService.GetCalendarNamesList()
	calendarNamesWithNull := enumWithNull(calendarNames)

	return agents.FunctionTool{
		Name:        "get_today_events",
		Description: "Get the user's schedule for today",
		ParamsJSONSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"calendar_name": map[string]any{
					"type":        []string{"string", "null"},
					"description": "Calendar name to get events from (optional - if omitted, gets from all calendars)",
					"enum":        calendarNamesWithNull,
				},
			},
			"additionalProperties": false,
			"required":             []string{"calendar_name"},
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return sa.handleGetTodayEvents(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}
}

// createGetWeekEventsTools creates the get this week's events tool
func (sa *ScheduleAgent) createGetWeekEventsTools() agents.FunctionTool {
	calendarNames := sa.calendarService.GetCalendarNamesList()
	calendarNamesWithNull := enumWithNull(calendarNames)

	return agents.FunctionTool{
		Name:        "get_week_events",
		Description: "Get the user's schedule for the week",
		ParamsJSONSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"calendar_name": map[string]any{
					"type":        []string{"string", "null"},
					"description": "Calendar name to get events from (optional - if omitted, gets from all calendars)",
					"enum":        calendarNamesWithNull,
				},
			},
			"required":             []string{"calendar_name"},
			"additionalProperties": false,
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return sa.handleGetWeekEvents(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}
}

// createGetSpecificDayEventsTools creates the get events for specific day tool
func (sa *ScheduleAgent) createGetSpecificDayEventsTools() agents.FunctionTool {
	calendarNames := sa.calendarService.GetCalendarNamesList()
	calendarNamesWithNull := enumWithNull(calendarNames)

	return agents.FunctionTool{
		Name:        "get_specific_day_events",
		Description: "Get the user's schedule for a specific date",
		ParamsJSONSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"date": map[string]any{
					"type":        "string",
					"description": "Date in YYYY-MM-DD format",
				},
				"calendar_name": map[string]any{
					"type":        []string{"string", "null"},
					"description": "Calendar name to get events from (optional - if omitted, gets from all calendars)",
					"enum":        calendarNamesWithNull,
				},
			},
			"additionalProperties": false,
			"required":             []string{"date", "calendar_name"},
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return sa.handleGetSpecificDayEvents(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}
}

// createCreateEventTools creates the create calendar event tool
func (sa *ScheduleAgent) createCreateEventTools() agents.FunctionTool {
	calendarNames := sa.calendarService.GetCalendarNamesList()

	return agents.FunctionTool{
		Name:        "create_calendar_event",
		Description: "Create a new calendar event",
		ParamsJSONSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title": map[string]any{
					"type":        "string",
					"description": "Title of the event",
				},
				"description": map[string]any{
					"type":        "string",
					"description": "Description of the event",
				},
				"start_time": map[string]any{
					"type":        "string",
					"description": "Start time in RFC3339 format (e.g., 2023-10-05T10:00:00Z)",
				},
				"end_time": map[string]any{
					"type":        "string",
					"description": "End time in RFC3339 format (e.g., 2023-10-05T11:00:00Z)",
				},
				"calendar_name": map[string]any{
					"type":        "string",
					"description": "Calendar name to create the event in",
					"enum":        calendarNames,
				},
			},
			"additionalProperties": false,
			"required":             []string{"title", "description", "start_time", "end_time", "calendar_name"},
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return sa.handleCreateEvent(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}
}

// createUpdateEventTools creates the update calendar event tool
func (sa *ScheduleAgent) createUpdateEventTools() agents.FunctionTool {
	calendarNames := sa.calendarService.GetCalendarNamesList()

	return agents.FunctionTool{
		Name:        "update_calendar_event",
		Description: "Update an existing calendar event",
		ParamsJSONSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"event_id": map[string]any{
					"type":        "string",
					"description": "ID of the event to update",
				},
				"calendar_name": map[string]any{
					"type":        "string",
					"description": "Calendar name where the event is located",
					"enum":        calendarNames,
				},
				"title": map[string]any{
					"type":        []string{"string", "null"},
					"description": "New title of the event (optional)",
				},
				"description": map[string]any{
					"type":        []string{"string", "null"},
					"description": "New description of the event (optional)",
				},
				"start_time": map[string]any{
					"type":        []string{"string", "null"},
					"description": "New start time in RFC3339 format (optional)",
				},
				"end_time": map[string]any{
					"type":        []string{"string", "null"},
					"description": "New end time in RFC3339 format (optional)",
				},
			},
			"additionalProperties": false,
			"required":             []string{"event_id", "calendar_name", "title", "description", "start_time", "end_time"},
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return sa.handleUpdateEvent(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}
}

// createDeleteEventTools creates the delete calendar event tool
func (sa *ScheduleAgent) createDeleteEventTools() agents.FunctionTool {
	calendarNames := sa.calendarService.GetCalendarNamesList()

	return agents.FunctionTool{
		Name:        "delete_calendar_event",
		Description: "Delete a calendar event",
		ParamsJSONSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"event_id": map[string]any{
					"type":        "string",
					"description": "ID of the event to delete",
				},
				"calendar_name": map[string]any{
					"type":        "string",
					"description": "Calendar name where the event is located",
					"enum":        calendarNames,
				},
			},
			"additionalProperties": false,
			"required":             []string{"event_id", "calendar_name"},
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return sa.handleDeleteEvent(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}
}

/** ---- TOOL HANDLERS ---- **/

// handleSearchEvents processes the search events tool invocation
func (sa *ScheduleAgent) handleSearchEvents(ctx context.Context, arguments string) (any, error) {
	if sa.ShouldDryRun(ctx) {
		return fmt.Sprintf("DRY RUN: Would search calendar events with args: %s", arguments), nil
	}

	arguments = normalizeArguments(arguments)

	// Parse arguments
	var args SearchEventsArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Validate fields
	if args.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	calendarName := refWithDefault(args.CalendarName, "")
	if !sa.calendarService.IsValidCalendarName(calendarName) {
		return nil, fmt.Errorf("invalid calendar name: %s", calendarName)
	}

	// Query events
	events, err := sa.calendarService.SearchEvents(ctx, args.Query, calendarName)
	if err != nil {
		return nil, fmt.Errorf("failed to search events: %w", err)
	}

	return formatEventsResponse(events), nil
}

// handleGetTodayEvents processes the get today's events tool invocation
func (sa *ScheduleAgent) handleGetTodayEvents(ctx context.Context, arguments string) (any, error) {
	if sa.ShouldDryRun(ctx) {
		return fmt.Sprintf("DRY RUN: Would get today's calendar events with args: %s", arguments), nil
	}

	arguments = normalizeArguments(arguments)

	// Parse arguments
	var args GetEventsArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Validate fields
	calendarName := refWithDefault(args.CalendarName, "")
	if !sa.calendarService.IsValidCalendarName(calendarName) {
		return nil, fmt.Errorf("invalid calendar name: %s", calendarName)
	}

	// Get today's events
	events, err := sa.calendarService.GetTodayEvents(ctx, calendarName)
	if err != nil {
		return nil, fmt.Errorf("failed to get today's events: %w", err)
	}

	return formatEventsResponse(events), nil
}

// handleGetWeekEvents processes the get week events tool invocation
func (sa *ScheduleAgent) handleGetWeekEvents(ctx context.Context, arguments string) (any, error) {
	if sa.ShouldDryRun(ctx) {
		return fmt.Sprintf("DRY RUN: Would get this week's calendar events with args: %s", arguments), nil
	}

	arguments = normalizeArguments(arguments)

	// Parse arguments
	var args GetEventsArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Validate fields
	calendarName := refWithDefault(args.CalendarName, "")
	if !sa.calendarService.IsValidCalendarName(calendarName) {
		return nil, fmt.Errorf("invalid calendar name: %s", calendarName)
	}

	// Get this week's events
	events, err := sa.calendarService.GetWeekEvents(ctx, calendarName)
	if err != nil {
		return nil, fmt.Errorf("failed to get week events: %w", err)
	}

	return formatEventsResponse(events), nil
}

// handleGetSpecificDayEvents processes the get specific day events tool invocation
func (sa *ScheduleAgent) handleGetSpecificDayEvents(ctx context.Context, arguments string) (any, error) {
	if sa.ShouldDryRun(ctx) {
		return fmt.Sprintf("DRY RUN: Would get specific day's calendar events with args: %s", arguments), nil
	}

	arguments = normalizeArguments(arguments)

	// Parse arguments
	var args GetSpecificDayEventsArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	calendarName := refWithDefault(args.CalendarName, "")

	// Validate fields
	if args.Date == "" {
		return nil, fmt.Errorf("date is required")
	}
	if !isValidDate(args.Date) {
		return nil, fmt.Errorf("invalid date format, expected YYYY-MM-DD")
	}
	if !sa.calendarService.IsValidCalendarName(calendarName) {
		return nil, fmt.Errorf("invalid calendar name: %s", calendarName)
	}

	if calendarName == "personal" {
		logger.Info("here")
	}

	// Parse date
	targetDate, err := time.Parse(gcal_service.DATE_FORMAT, args.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
	}

	// Get events for the specific day
	start := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())
	end := start.Add(24 * time.Hour)

	events, err := sa.calendarService.GetEventsForTimeRange(ctx, start, end, calendarName)
	if err != nil {
		return nil, fmt.Errorf("failed to get events for specific day: %w", err)
	}

	return formatEventsResponse(events), nil
}

// handleCreateEvent processes the create event tool invocation
func (sa *ScheduleAgent) handleCreateEvent(ctx context.Context, arguments string) (any, error) {
	if sa.ShouldDryRun(ctx) {
		return fmt.Sprintf("DRY RUN: Would create calendar event with args: %s", arguments), nil
	}

	arguments = normalizeArguments(arguments)

	// Parse arguments
	var args CreateEventArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Validate required fields
	if args.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if args.StartTime == "" {
		return nil, fmt.Errorf("start_time is required")
	}
	if args.EndTime == "" {
		return nil, fmt.Errorf("end_time is required")
	}
	if args.CalendarName == "" {
		return nil, fmt.Errorf("calendar_name is required")
	}

	event, err := sa.calendarService.CreateEvent(ctx, gcal_service.CreateEventInput{
		Title:        args.Title,
		Description:  args.Description,
		Start:        args.StartTime,
		End:          args.EndTime,
		CalendarName: args.CalendarName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	return formatEventResponse(event), nil
}

// handleUpdateEvent processes the update event tool invocation
func (sa *ScheduleAgent) handleUpdateEvent(ctx context.Context, arguments string) (any, error) {
	if sa.ShouldDryRun(ctx) {
		return fmt.Sprintf("DRY RUN: Would update calendar event with args: %s", arguments), nil
	}

	arguments = normalizeArguments(arguments)

	// Parse arguments
	var args UpdateEventArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Validate required fields
	if args.EventID == "" {
		return nil, fmt.Errorf("event_id is required")
	}
	if args.CalendarName == "" {
		return nil, fmt.Errorf("calendar_name is required")
	}

	event, err := sa.calendarService.UpdateEvent(ctx, gcal_service.UpdateEventInput{
		EventID:      args.EventID,
		CalendarName: args.CalendarName,
		Title:        args.Title,
		Description:  args.Description,
		Start:        args.StartTime,
		End:          args.EndTime,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	return formatEventResponse(event), nil
}

// handleDeleteEvent processes the delete event tool invocation
func (sa *ScheduleAgent) handleDeleteEvent(ctx context.Context, arguments string) (any, error) {
	if sa.ShouldDryRun(ctx) {
		return fmt.Sprintf("DRY RUN: Would delete calendar event with args: %s", arguments), nil
	}

	arguments = normalizeArguments(arguments)

	// Parse arguments
	var args DeleteEventArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Validate required fields
	if args.EventID == "" {
		return nil, fmt.Errorf("event_id is required")
	}
	if args.CalendarName == "" {
		return nil, fmt.Errorf("calendar_name is required")
	}

	if !sa.calendarService.IsValidCalendarName(args.CalendarName) {
		return nil, fmt.Errorf("invalid calendar name: %s", args.CalendarName)
	}

	err := sa.calendarService.DeleteEvent(ctx, gcal_service.DeleteEventInput{
		EventID:      args.EventID,
		CalendarName: args.CalendarName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to delete event: %w", err)
	}

	return map[string]any{
		"success":       true,
		"message":       "Event deleted successfully",
		"event_id":      args.EventID,
		"calendar_name": args.CalendarName,
	}, nil
}

// Helper function to get a value with default
func refWithDefault[T any](v *T, d T) T {
	if v == nil {
		return d
	}
	return *v
}

// enumWithNull appends a JSON null option to a string enum.
func enumWithNull(values []string) []any {
	enumValues := make([]any, 0, len(values)+1)
	for _, v := range values {
		enumValues = append(enumValues, v)
	}
	enumValues = append(enumValues, nil)

	return enumValues
}
