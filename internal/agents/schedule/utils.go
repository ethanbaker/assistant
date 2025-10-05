package schedule

import (
	"fmt"
	"strings"
	"time"

	"google.golang.org/api/calendar/v3"
)

// getCalendarNamesList returns a list of calendar names for enum validation
func (sa *ScheduleAgent) getCalendarNamesList() []string {
	if sa.calendarService == nil || len(sa.calendarService.calendarConfig.Calendars) == 0 {
		return []string{"primary"}
	}

	names := make([]string, 0, len(sa.calendarService.calendarConfig.Calendars))
	for _, cal := range sa.calendarService.calendarConfig.Calendars {
		names = append(names, cal.Name)
	}

	// Always include primary as an option
	hasPrimary := false
	for _, name := range names {
		if strings.EqualFold(name, "primary") {
			hasPrimary = true
			break
		}
	}
	if !hasPrimary {
		names = append(names, "primary")
	}

	return names
}

// isValidCalendarName validates that the calendar name exists in the configured calendars
func (sa *ScheduleAgent) isValidCalendarName(calendarName string) bool {
	// If no calendars are configured, return false
	if sa.calendarService == nil || len(sa.calendarService.calendarConfig.Calendars) == 0 {
		return false
	}

	// Allow empty calendar name
	if calendarName == "" {
		return true
	}

	// Check configured calendars
	for _, cal := range sa.calendarService.calendarConfig.Calendars {
		if strings.EqualFold(cal.Name, calendarName) {
			return true
		}
	}

	return false
}

// isValidDate checks if a date string is in the correct format
func isValidDate(dateStr string) bool {
	// Allow empty dates
	if dateStr == "" {
		return true
	}

	_, err := time.Parse(DATE_FORMAT, dateStr)
	return err == nil
}

// formatEventsResponse formats a list of events for the response
func (sa *ScheduleAgent) formatEventsResponse(events []*calendar.Event) any {
	// Handle no events case
	if len(events) == 0 {
		return map[string]any{
			"message": "No events found",
			"events":  []map[string]any{},
		}
	}

	// Format each event
	formattedEvents := make([]map[string]any, len(events))
	for i, event := range events {
		formattedEvents[i] = sa.formatEventResponse(event)
	}

	return map[string]any{
		"message": fmt.Sprintf("Found %d event(s)", len(events)),
		"events":  formattedEvents,
	}
}

// formatEventResponse formats a single event for the response
func (sa *ScheduleAgent) formatEventResponse(event *calendar.Event) map[string]any {
	eventData := map[string]any{
		"id":          event.Id,
		"title":       event.Summary,
		"description": event.Description,
		"organizer":   event.Organizer,
		"attendees":   []map[string]any{},
	}

	// Format attendees
	if event.Attendees != nil {
		attendees := make([]map[string]any, len(event.Attendees))
		for i, attendee := range event.Attendees {
			attendeeData := map[string]any{
				"email":          attendee.Email,
				"displayName":    attendee.DisplayName,
				"responseStatus": attendee.ResponseStatus,
			}
			attendees[i] = attendeeData
		}
		eventData["attendees"] = attendees
	}

	// Format start time
	if event.Start != nil {
		if event.Start.DateTime != "" {
			eventData["start_time"] = event.Start.DateTime
			eventData["all_day"] = false
		} else if event.Start.Date != "" {
			eventData["start_time"] = event.Start.Date
			eventData["all_day"] = true
		}
	}

	// Format end time
	if event.End != nil {
		if event.End.DateTime != "" {
			eventData["end_time"] = event.End.DateTime
		} else if event.End.Date != "" {
			eventData["end_time"] = event.End.Date
		}
	}

	// Add location if available
	if event.Location != "" {
		eventData["location"] = event.Location
	}

	return eventData
}
