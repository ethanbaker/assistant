package schedule

import (
	"fmt"

	gcal "github.com/ethanbaker/assistant/internal/services/gcal"
)

// formatEventsResponse formats a list of gcal events for the tool response.
func formatEventsResponse(events []*gcal.Event) any {
	if len(events) == 0 {
		return map[string]any{
			"message": "No events found",
			"events":  []map[string]any{},
		}
	}

	formattedEvents := make([]map[string]any, len(events))
	for i, event := range events {
		formattedEvents[i] = formatEventResponse(event)
	}

	return map[string]any{
		"message": fmt.Sprintf("Found %d event(s)", len(events)),
		"events":  formattedEvents,
	}
}

// formatEventResponse formats a single gcal event for the tool response.
func formatEventResponse(event *gcal.Event) map[string]any {
	if event == nil {
		return map[string]any{}
	}

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
