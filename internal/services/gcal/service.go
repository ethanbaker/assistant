package gcal

import (
	"context"
	"time"
)

// EventDateTime holds start or end time for an event (datetime or date-only for all-day).
type EventDateTime struct {
	DateTime string `json:"date_time,omitempty"`
	Date     string `json:"date,omitempty"`
	TimeZone string `json:"time_zone,omitempty"`
}

// EventAttendee represents a calendar event attendee.
type EventAttendee struct {
	Email          string `json:"email,omitempty"`
	DisplayName    string `json:"display_name,omitempty"`
	ResponseStatus string `json:"response_status,omitempty"`
}

// EventOrganizer represents the event organizer.
type EventOrganizer struct {
	Email       string `json:"email,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Self        bool   `json:"self,omitempty"`
}

// Event is the gcal service event type, a superset of the Google Calendar API event.
// Agents depend on this type instead of calendar.Event.
type Event struct {
	Id          string           `json:"id,omitempty"`
	Summary     string           `json:"summary,omitempty"`
	Description string           `json:"description,omitempty"`
	Location    string           `json:"location,omitempty"`
	Start       *EventDateTime   `json:"start,omitempty"`
	End         *EventDateTime   `json:"end,omitempty"`
	Attendees   []*EventAttendee `json:"attendees,omitempty"`
	Organizer   *EventOrganizer  `json:"organizer,omitempty"`
}

// CreateEventInput is the input for creating a calendar event.
type CreateEventInput struct {
	Title        string
	Description  string
	Start        string // RFC3339
	End          string // RFC3339
	CalendarName string
}

// UpdateEventInput is the input for updating a calendar event. Optional fields are pointers; nil means no change.
type UpdateEventInput struct {
	EventID      string
	CalendarName string
	Title        *string
	Description  *string
	Start        *string // RFC3339
	End          *string // RFC3339
}

// DeleteEventInput is the input for deleting a calendar event.
type DeleteEventInput struct {
	EventID      string
	CalendarName string
}

// CalendarService defines the gcal service interface.
type CalendarService interface {
	SearchEvents(ctx context.Context, query string, calendarName string) ([]*Event, error)
	GetEventsForTimeRange(ctx context.Context, start, end time.Time, calendarName string) ([]*Event, error)
	GetTodayEvents(ctx context.Context, calendarName string) ([]*Event, error)
	GetWeekEvents(ctx context.Context, calendarName string) ([]*Event, error)
	CreateEvent(ctx context.Context, in CreateEventInput) (*Event, error)
	UpdateEvent(ctx context.Context, in UpdateEventInput) (*Event, error)
	DeleteEvent(ctx context.Context, in DeleteEventInput) error
	GetCalendarNamesList() []string
	GetCalendars() []Calendar
	IsValidCalendarName(calendarName string) bool
	GetTimezone() *time.Location
}

// Ensure calendarServiceImpl implements CalendarService
var _ CalendarService = (*calendarServiceImpl)(nil)
