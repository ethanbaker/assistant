package gcal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// Calendar holds user provided calendar information
type Calendar struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
	ID          string `json:"id" yaml:"id"`
}

// AccountConfig holds one config entry per Google calendar account
type AccountConfig struct {
	TokenFilePath string     `json:"token_path" yaml:"token_path"`
	Calendars     []Calendar `json:"calendars" yaml:"calendars"`
}

// CalendarServiceConfig provided on construct
type CalendarServiceConfig struct {
	CredentialsFilePath string          `json:"credentials_path" yaml:"credentials_path"`
	Timezone            string          `json:"timezone" yaml:"timezone"`
	Accounts            []AccountConfig `json:"accounts" yaml:"accounts"`
}

// calendarServiceImpl wraps the Google Calendar API service and implements CalendarService.
type calendarServiceImpl struct {
	calendars    []Calendar                   // User provided calendars
	services     map[string]*calendar.Service // Map of calendar name -> associated google calendar service
	tokenSources []*tokenSavingSource         // List of active tokens

	timezone        *time.Location // Timezone to standardize events
	credentialsPath string         // Filepath to app credentials
}

// NewCalendarService creates a new CalendarService instance
func NewCalendarService(cfg CalendarServiceConfig) (CalendarService, error) {
	if cfg.CredentialsFilePath == "" {
		return nil, fmt.Errorf("CredentialsFilePath is required")
	}
	if cfg.Timezone == "" {
		return nil, fmt.Errorf("Timezone is required")
	}

	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return nil, fmt.Errorf("Failed to load timezone '%s': %v", cfg.Timezone, err)
	}

	calendarService := &calendarServiceImpl{
		credentialsPath: cfg.CredentialsFilePath,
		timezone:        loc,
		services:        map[string]*calendar.Service{},
	}

	for _, account := range cfg.Accounts {
		if err := calendarService.setupAccount(account); err != nil {
			return nil, err
		}
	}

	return calendarService, nil
}

// setupAccount is a helper method to add accounts to the calendar service
func (cs *calendarServiceImpl) setupAccount(cfg AccountConfig) error {
	if cfg.TokenFilePath == "" {
		return fmt.Errorf("TokenFilePath is required")
	}
	if len(cfg.Calendars) == 0 {
		return fmt.Errorf("Calendars slice cannot be empty")
	}

	// Read credentials JSON file
	credentialsJSON, err := os.ReadFile(cs.credentialsPath)
	if err != nil {
		return fmt.Errorf("failed to read credentials file: %w", err)
	}

	// Read and parse OAuth2 token JSON file
	tokenJSON, err := os.ReadFile(cfg.TokenFilePath)
	if err != nil {
		return fmt.Errorf("failed to read token file: %w", err)
	}

	var token oauth2.Token
	err = json.Unmarshal(tokenJSON, &token)
	if err != nil {
		return fmt.Errorf("failed to parse token JSON: %w", err)
	}

	// Parse credentials to get OAuth2 config
	config, err := google.ConfigFromJSON(credentialsJSON, calendar.CalendarScope)
	if err != nil {
		return fmt.Errorf("failed to parse credentials: %w", err)
	}

	// Create a token source that automatically refreshes the token
	tokenSource := config.TokenSource(context.Background(), &token)

	// Wrap the token source to save tokens when they're refreshed
	savingTokenSource := &tokenSavingSource{
		source:    tokenSource,
		tokenPath: cfg.TokenFilePath,
	}

	// Get a fresh token (this will refresh if needed)
	freshToken, err := savingTokenSource.Token()
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// If the token was refreshed, save it back to the file
	if freshToken.AccessToken != token.AccessToken {
		err = saveToken(cfg.TokenFilePath, freshToken)
		if err != nil {
			return fmt.Errorf("failed to save refreshed token: %w", err)
		}
	}

	// Create OAuth2 calendar client with the token source
	client := oauth2.NewClient(context.Background(), savingTokenSource)
	service, err := calendar.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("failed to create calendar service: %w", err)
	}

	// Add to existing calendar service
	for _, c := range cfg.Calendars {
		cs.calendars = append(cs.calendars, c)
		cs.services[c.Name] = service
	}
	cs.tokenSources = append(cs.tokenSources, savingTokenSource)

	return nil
}

// fromCalendarEvent converts a Google Calendar API event to the gcal Event type.
func (cs *calendarServiceImpl) fromCalendarEvent(e *calendar.Event) *Event {
	if e == nil {
		return nil
	}
	ev := &Event{
		Id:          e.Id,
		Summary:     e.Summary,
		Description: e.Description,
		Location:    e.Location,
	}
	if e.Start != nil {
		ev.Start = &EventDateTime{
			DateTime: e.Start.DateTime,
			Date:     e.Start.Date,
			TimeZone: e.Start.TimeZone,
		}
	}
	if e.End != nil {
		ev.End = &EventDateTime{
			DateTime: e.End.DateTime,
			Date:     e.End.Date,
			TimeZone: e.End.TimeZone,
		}
	}
	if e.Organizer != nil {
		ev.Organizer = &EventOrganizer{
			Email:       e.Organizer.Email,
			DisplayName: e.Organizer.DisplayName,
			Self:        e.Organizer.Self,
		}
	}
	if e.Attendees != nil {
		ev.Attendees = make([]*EventAttendee, len(e.Attendees))
		for i, a := range e.Attendees {
			ev.Attendees[i] = &EventAttendee{
				Email:          a.Email,
				DisplayName:    a.DisplayName,
				ResponseStatus: a.ResponseStatus,
			}
		}
	}
	return ev
}

// SearchEvents searches for events by name and optional calendar name
func (cs *calendarServiceImpl) SearchEvents(ctx context.Context, query string, calendarName string) ([]*Event, error) {
	names := cs.getCalendarNames(calendarName)

	// Collect events from all specified calendars
	rawEvents := []*calendar.Event{}
	for _, n := range names {
		id := cs.getCalendarID(n)

		service, ok := cs.services[n]
		if !ok || service == nil {
			return nil, fmt.Errorf("calendar %s does not have an associated service", n)
		}

		call := service.Events.List(id).
			Q(query).                            // Search query
			SingleEvents(true).                  // Expand recurring events
			OrderBy("startTime").                // Order by start time
			MaxResults(GOOGLE_EVENT_MAX_RESULTS) // Limit results

		evs, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to search events: %w", err)
		}

		rawEvents = append(rawEvents, evs.Items...)
	}

	// Sort events so that start time is descending
	var j int
	for i := 1; i < len(rawEvents); i++ {
		e := rawEvents[i]

		// Parse pivot event time
		t1, err := time.Parse(time.RFC3339, e.Start.DateTime)
		if err != nil {
			continue
		}

		for j = i - 1; j >= 0; j-- {
			// Parse comparison event time
			t0, err := time.Parse(time.RFC3339, rawEvents[j].Start.DateTime)
			if err != nil || t1.Before(t0) {
				break
			}
			rawEvents[j+1] = rawEvents[j]
		}

		rawEvents[j+1] = e
	}

	// Trim for AI context
	if len(rawEvents) > AI_CONTEXT_MAX_EVENTS {
		rawEvents = rawEvents[:AI_CONTEXT_MAX_EVENTS]
	}

	// Standardize timezone
	for _, e := range rawEvents {
		if err := cs.fixTimezone(e.Start); err != nil {
			return nil, err
		}
		if err := cs.fixTimezone(e.End); err != nil {
			return nil, err
		}
	}

	events := make([]*Event, len(rawEvents))
	for i, e := range rawEvents {
		events[i] = cs.fromCalendarEvent(e)
	}
	return events, nil
}

// GetEventsForTimeRange gets events for a specific time range. If calendarName is empty, fetch from all calendars
func (cs *calendarServiceImpl) GetEventsForTimeRange(ctx context.Context, start, end time.Time, calendarName string) ([]*Event, error) {
	names := cs.getCalendarNames(calendarName)

	rawEvents := []*calendar.Event{}
	for _, n := range names {
		// Fetch events for each calendar
		id := cs.getCalendarID(n)

		service, ok := cs.services[n]
		if !ok || service == nil {
			return nil, fmt.Errorf("calendar %s does not have an associated service", n)
		}

		call := service.Events.List(id).
			TimeMin(start.Format(time.RFC3339)).
			TimeMax(end.Format(time.RFC3339)).
			SingleEvents(true).
			OrderBy("startTime")

		evs, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to get events: %w", err)
		}

		rawEvents = append(rawEvents, evs.Items...)
	}

	// Sort events so that start time is descending
	var j int
	for i := 1; i < len(rawEvents); i++ {
		e := rawEvents[i]

		// Parse pivot event time
		t0, err := time.Parse(time.RFC3339, e.Start.DateTime)
		if err != nil {
			continue
		}

		for j = i - 1; j >= 0; j-- {
			// Parse comparison event time
			t1, err := time.Parse(time.RFC3339, rawEvents[j].Start.DateTime)
			if err != nil || t0.After(t1) {
				break
			}
			rawEvents[j+1] = rawEvents[j]
		}

		rawEvents[j+1] = e
	}

	// Standardize timezone
	for _, e := range rawEvents {
		if err := cs.fixTimezone(e.Start); err != nil {
			return nil, err
		}
		if err := cs.fixTimezone(e.End); err != nil {
			return nil, err
		}
	}

	events := make([]*Event, len(rawEvents))
	for i, e := range rawEvents {
		events[i] = cs.fromCalendarEvent(e)
	}
	return events, nil
}

// GetTodayEvents gets events for today. If calendarName is empty, fetch all calendars
func (cs *calendarServiceImpl) GetTodayEvents(ctx context.Context, calendarName string) ([]*Event, error) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.Add(24 * time.Hour)

	names := cs.getCalendarNames(calendarName)

	events := []*Event{}
	for _, n := range names {
		evs, err := cs.GetEventsForTimeRange(ctx, start, end, n)
		if err != nil {
			return nil, err
		}
		events = append(events, evs...)
	}

	return events, nil
}

// GetWeekEvents gets events for this week. If calendarName is empty, fetch all calendars
func (cs *calendarServiceImpl) GetWeekEvents(ctx context.Context, calendarName string) ([]*Event, error) {
	now := time.Now()
	weekday := int(now.Weekday())
	start := now.AddDate(0, 0, -weekday) // Start of week (Sunday)
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	end := start.Add(7 * 24 * time.Hour)

	names := cs.getCalendarNames(calendarName)

	events := []*Event{}
	for _, n := range names {
		evs, err := cs.GetEventsForTimeRange(ctx, start, end, n)
		if err != nil {
			return nil, err
		}
		events = append(events, evs...)
	}

	return events, nil
}

// CreateEvent creates a new calendar event
func (cs *calendarServiceImpl) CreateEvent(ctx context.Context, in CreateEventInput) (*Event, error) {
	calendarID := cs.getCalendarID(in.CalendarName)
	if calendarID == "" {
		return nil, fmt.Errorf("invalid calendar name: %s", in.CalendarName)
	}

	// Parse times
	startTimeInput, err := time.Parse(time.RFC3339, in.Start)
	if err != nil {
		return nil, fmt.Errorf("invalid start_time format, expected RFC3339: %w", err)
	}

	endTimeInput, err := time.Parse(time.RFC3339, in.End)
	if err != nil {
		return nil, fmt.Errorf("invalid end_time format, expected RFC3339: %w", err)
	}

	// Create start and end times in the standard timezone
	st := time.Date(startTimeInput.Year(), startTimeInput.Month(), startTimeInput.Day(), startTimeInput.Hour(), startTimeInput.Minute(), startTimeInput.Second(), startTimeInput.Nanosecond(), cs.timezone)
	et := time.Date(endTimeInput.Year(), endTimeInput.Month(), endTimeInput.Day(), endTimeInput.Hour(), endTimeInput.Minute(), endTimeInput.Second(), endTimeInput.Nanosecond(), cs.timezone)

	// Validate that end time is after start time
	if et.Before(st) || et.Equal(st) {
		return nil, fmt.Errorf("end_time must be after start_time")
	}

	event := &calendar.Event{
		Summary:     in.Title,
		Description: in.Description,
		Start: &calendar.EventDateTime{
			DateTime: st.Format(time.RFC3339),
			TimeZone: cs.timezone.String(),
		},
		End: &calendar.EventDateTime{
			DateTime: et.Format(time.RFC3339),
			TimeZone: cs.timezone.String(),
		},
	}

	service, ok := cs.services[in.CalendarName]
	if !ok || service == nil {
		return nil, fmt.Errorf("calendar %s does not have an associated service", in.CalendarName)
	}

	createdEvent, err := service.Events.Insert(calendarID, event).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	// Fix any timezone format issues
	if err := cs.fixTimezone(createdEvent.Start); err != nil {
		return nil, err
	}
	if err := cs.fixTimezone(createdEvent.End); err != nil {
		return nil, err
	}

	return cs.fromCalendarEvent(createdEvent), nil
}

// UpdateEvent updates an existing calendar event
func (cs *calendarServiceImpl) UpdateEvent(ctx context.Context, in UpdateEventInput) (*Event, error) {
	calendarID := cs.getCalendarID(in.CalendarName)
	if calendarID == "" {
		return nil, fmt.Errorf("invalid calendar name: %s", in.CalendarName)
	}

	service, ok := cs.services[in.CalendarName]
	if !ok || service == nil {
		return nil, fmt.Errorf("calendar %s does not have an associated service", in.CalendarName)
	}

	// First get the existing event
	existingEvent, err := service.Events.Get(calendarID, in.EventID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get existing event: %w", err)
	}

	// Parse times if provided
	var st, et time.Time
	start := ""
	end := ""
	if in.Start != nil {
		start = *in.Start
	}
	if in.End != nil {
		end = *in.End
	}

	if start != "" {
		t, err := time.Parse(time.RFC3339, start)
		if err != nil {
			return nil, fmt.Errorf("invalid start_time format, expected RFC3339: %w", err)
		}
		st = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), cs.timezone)
	}

	if end != "" {
		t, err := time.Parse(time.RFC3339, end)
		if err != nil {
			return nil, fmt.Errorf("invalid end_time format, expected RFC3339: %w", err)
		}
		et = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), cs.timezone)
	}

	// Validate that end time is after start time if both are provided
	if start != "" && end != "" {
		if et.Before(st) || et.Equal(st) {
			return nil, fmt.Errorf("end_time must be after start_time")
		}
	}

	// Update the fields
	if in.Title != nil && *in.Title != "" {
		existingEvent.Summary = *in.Title
	}
	if in.Description != nil {
		existingEvent.Description = *in.Description
	}
	if start != "" {
		existingEvent.Start = &calendar.EventDateTime{
			DateTime: st.Format(time.RFC3339),
			TimeZone: cs.timezone.String(),
		}
	}
	if end != "" {
		existingEvent.End = &calendar.EventDateTime{
			DateTime: et.Format(time.RFC3339),
			TimeZone: cs.timezone.String(),
		}
	}

	updatedEvent, err := service.Events.Update(calendarID, in.EventID, existingEvent).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	// Fix any timezone format issues
	if err := cs.fixTimezone(updatedEvent.Start); err != nil {
		return nil, err
	}
	if err := cs.fixTimezone(updatedEvent.End); err != nil {
		return nil, err
	}

	return cs.fromCalendarEvent(updatedEvent), nil
}

// DeleteEvent deletes a calendar event
func (cs *calendarServiceImpl) DeleteEvent(ctx context.Context, in DeleteEventInput) error {
	calendarID := cs.getCalendarID(in.CalendarName)
	if calendarID == "" {
		return fmt.Errorf("invalid calendar name: %s", in.CalendarName)
	}

	service, ok := cs.services[in.CalendarName]
	if !ok || service == nil {
		return fmt.Errorf("calendar %s does not have an associated service", in.CalendarName)
	}

	err := service.Events.Delete(calendarID, in.EventID).Do()
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	return nil
}

// GetCalendarNamesList returns a list of calendar names for enum validation
func (cs *calendarServiceImpl) GetCalendarNamesList() []string {
	if len(cs.calendars) == 0 {
		return []string{"primary"}
	}

	names := make([]string, 0, len(cs.calendars))
	for _, cal := range cs.calendars {
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

// GetCalendars returns a list of calendars
func (cs *calendarServiceImpl) GetCalendars() []Calendar {
	return cs.calendars
}

// GetTimezone returns the services timezone
func (cs *calendarServiceImpl) GetTimezone() *time.Location {
	return cs.timezone
}

// IsValidCalendarName validates that the calendar name exists in the configured calendars
func (cs *calendarServiceImpl) IsValidCalendarName(calendarName string) bool {
	// If no calendars are configured, return false
	if len(cs.calendars) == 0 {
		return false
	}

	// Allow empty calendar name
	if calendarName == "" {
		return true
	}

	// Check configured calendars
	for _, cal := range cs.calendars {
		if strings.EqualFold(cal.Name, calendarName) {
			return true
		}
	}

	return false
}

// Helper function to map calendar name to IDs
func (cs *calendarServiceImpl) getCalendarID(calendarName string) string {
	// Map calendar name to actual calendar IDs
	if calendarName != "" {
		for _, cal := range cs.calendars {
			if strings.EqualFold(cal.Name, calendarName) {
				return cal.ID
			}
		}
	}
	return ""
}

// Helper function to find calendar names
func (cs *calendarServiceImpl) getCalendarNames(input string) []string {
	names := []string{}
	if input == "" {
		// If input is empty, fetch events from all calendars
		for _, c := range cs.calendars {
			names = append(names, c.Name)
		}
	} else {
		// Otherwise, just use the specified input
		names = append(names, input)
	}

	return names
}

// Helper function to format times to the service's timezone
func (cs *calendarServiceImpl) fixTimezone(event *calendar.EventDateTime) error {
	if event == nil || event.DateTime == "" {
		return nil
	}

	// Parse time
	t, err := time.Parse(time.RFC3339, event.DateTime)
	if err != nil {
		return err
	}

	// Create new time object in the right timezone
	nt := t.In(cs.timezone)

	// Update event
	event.DateTime = nt.Format(time.RFC3339)
	event.Date = nt.Format("yyyy-MM-dd")
	event.TimeZone = cs.timezone.String()

	return nil
}
