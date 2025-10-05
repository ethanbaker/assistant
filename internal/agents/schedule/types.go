package schedule

import (
	"fmt"
	"time"

	"google.golang.org/api/calendar/v3"
)

// CalendarEvent represents a calendar event in a standardized format for ai models to use
type CalendarEvent struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Calendar    string    `json:"calendar"`
	AllDay      bool      `json:"all_day"`
}

// ConvertGoogleEvent converts a Google Calendar event to our internal format
func ConvertGoogleEvent(gcalEvent *calendar.Event) (*CalendarEvent, error) {
	event := &CalendarEvent{
		ID:          gcalEvent.Id,
		Title:       gcalEvent.Summary,
		Description: gcalEvent.Description,
	}

	// Parse start time
	if gcalEvent.Start != nil {
		if gcalEvent.Start.DateTime != "" {
			// DateTime event
			startTime, err := time.Parse(time.RFC3339, gcalEvent.Start.DateTime)
			if err != nil {
				return nil, fmt.Errorf("failed to parse start time: %w", err)
			}
			event.StartTime = startTime
		} else if gcalEvent.Start.Date != "" {
			// All-day event
			startTime, err := time.Parse("2006-01-02", gcalEvent.Start.Date)
			if err != nil {
				return nil, fmt.Errorf("failed to parse start date: %w", err)
			}
			event.StartTime = startTime
			event.AllDay = true
		}
	}

	// Parse end time
	if gcalEvent.End != nil {
		if gcalEvent.End.DateTime != "" {
			endTime, err := time.Parse(time.RFC3339, gcalEvent.End.DateTime)
			if err != nil {
				return nil, fmt.Errorf("failed to parse end time: %w", err)
			}
			event.EndTime = endTime
		} else if gcalEvent.End.Date != "" {
			endTime, err := time.Parse("2006-01-02", gcalEvent.End.Date)
			if err != nil {
				return nil, fmt.Errorf("failed to parse end date: %w", err)
			}
			event.EndTime = endTime
		}
	}

	return event, nil
}

// ConvertMultipleEvents converts multiple Google Calendar events
func ConvertMultipleEvents(gcalEvents []*calendar.Event) ([]*CalendarEvent, error) {
	events := make([]*CalendarEvent, 0, len(gcalEvents))

	for _, gcalEvent := range gcalEvents {
		event, err := ConvertGoogleEvent(gcalEvent)
		if err != nil {
			// Log error but continue processing other events
			continue
		}
		events = append(events, event)
	}

	return events, nil
}
