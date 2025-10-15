package schedule

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ethanbaker/assistant/pkg/utils"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v3"
)

const (
	GOOGLE_EVENT_MAX_RESULTS = 500
	AI_CONTEXT_MAX_EVENTS    = 20
)

// tokenSavingSource wraps an oauth2.TokenSource and automatically saves
// refreshed tokens to disk
type tokenSavingSource struct {
	source    oauth2.TokenSource
	tokenPath string
	lastToken *oauth2.Token
}

// Token returns a valid token, refreshing if necessary and saving to disk
func (t *tokenSavingSource) Token() (*oauth2.Token, error) {
	token, err := t.source.Token()
	if err != nil {
		return nil, err
	}

	// If this is a new token (different access token), save it
	if t.lastToken == nil || t.lastToken.AccessToken != token.AccessToken {
		if saveErr := saveToken(t.tokenPath, token); saveErr != nil {
			// Log the error but don't fail the request
			fmt.Fprintf(os.Stderr, "Warning: failed to save refreshed token: %v\n", saveErr)
		}
		t.lastToken = token
	}

	return token, nil
}

// CalendarConfig represents the structure of the calendar configuration file
type CalendarConfig struct {
	Calendars []struct {
		Name        string `json:"name" yaml:"name"`
		Description string `json:"description" yaml:"description"`
		ID          string `json:"id" yaml:"id"`
	} `json:"calendars" yaml:"calendars"`
}

// CalendarService wraps the Google Calendar API service
type CalendarService struct {
	service        *calendar.Service
	cfg            *utils.Config
	calendarConfig CalendarConfig
	tokenSource    oauth2.TokenSource
	tokenPath      string
}

// NewCalendarService creates a new CalendarService instance
func NewCalendarService(ctx context.Context, cfg *utils.Config) (*CalendarService, error) {
	// Read env variables for credentials and token paths
	credentialsPath := cfg.Get("GOOGLE_CALENDAR_CREDENTIALS_JSON")
	if credentialsPath == "" {
		return nil, fmt.Errorf("GOOGLE_CALENDAR_CREDENTIALS_JSON not set in environment")
	}

	tokenPath := cfg.Get("GOOGLE_CALENDAR_TOKEN_JSON")
	if tokenPath == "" {
		return nil, fmt.Errorf("GOOGLE_CALENDAR_TOKEN_JSON not set in environment")
	}

	// Read credentials JSON file
	credentialsJSON, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	// Read and parse OAuth2 token JSON file
	tokenJSON, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	var token oauth2.Token
	err = json.Unmarshal(tokenJSON, &token)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token JSON: %w", err)
	}

	// Parse credentials to get OAuth2 config
	config, err := google.ConfigFromJSON(credentialsJSON, calendar.CalendarScope)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	// Create a token source that automatically refreshes the token
	tokenSource := config.TokenSource(ctx, &token)
	
	// Wrap the token source to save tokens when they're refreshed
	savingTokenSource := &tokenSavingSource{
		source:    tokenSource,
		tokenPath: tokenPath,
	}
	
	// Get a fresh token (this will refresh if needed)
	freshToken, err := savingTokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	// If the token was refreshed, save it back to the file
	if freshToken.AccessToken != token.AccessToken {
		err = saveToken(tokenPath, freshToken)
		if err != nil {
			return nil, fmt.Errorf("failed to save refreshed token: %w", err)
		}
	}

	// Create OAuth2 calendar client with the token source
	client := oauth2.NewClient(ctx, savingTokenSource)
	service, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar service: %w", err)
	}

	// Initialize calendar IDs map from config
	calendarConfigPath := cfg.Get("GOOGLE_CALENDARS_CONFIG")
	if calendarConfigPath == "" {
		return nil, fmt.Errorf("GOOGLE_CALENDARS_CONFIG not set in environment")
	}

	f, err := os.ReadFile(calendarConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read calendar config file: %w", err)
	}

	var calConfig CalendarConfig
	err = yaml.Unmarshal(f, &calConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load calendar config: %w", err)
	}

	return &CalendarService{
		service:        service,
		cfg:            cfg,
		calendarConfig: calConfig,
		tokenSource:    savingTokenSource,
		tokenPath:      tokenPath,
	}, nil
}

// SearchEvents searches for events by name and optional calendar name
func (cs *CalendarService) SearchEvents(ctx context.Context, query string, calendarName string) ([]*calendar.Event, error) {
	calendarNames := []string{}
	if calendarName == "" {
		// If calendarName is empty, search in all calendars
		for _, cal := range cs.calendarConfig.Calendars {
			calendarNames = append(calendarNames, cal.Name)
		}
	} else {
		// Otherwise, just use the specified calendar
		calendarNames = append(calendarNames, calendarName)
	}

	// Collect events from all specified calendars
	allEvents := []*calendar.Event{}
	for _, calName := range calendarNames {
		calendarID := cs.getCalendarID(calName)

		call := cs.service.Events.List(calendarID).
			Q(query).                            // Search query
			SingleEvents(true).                  // Expand recurring events
			OrderBy("startTime").                // Order by start time
			MaxResults(GOOGLE_EVENT_MAX_RESULTS) // Limit results

		events, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to search events: %w", err)
		}

		allEvents = append(allEvents, events.Items...)
	}

	// Sort events so that start time is descending
	var j int
	for i := 1; i < len(allEvents); i++ {
		event := allEvents[i]

		// Parse pivot event time
		t1, err := time.Parse(time.RFC3339, event.Start.DateTime)
		if err != nil {
			continue
		}

		for j = i - 1; j >= 0; j-- {
			// Parse comparison event time
			t0, err := time.Parse(time.RFC3339, allEvents[j].Start.DateTime)
			if err != nil || t1.Before(t0) {
				break
			}
			allEvents[j+1] = allEvents[j]
		}

		allEvents[j+1] = event
	}

	// Trim for AI context
	if len(allEvents) > AI_CONTEXT_MAX_EVENTS {
		allEvents = allEvents[:AI_CONTEXT_MAX_EVENTS]
	}

	return allEvents, nil
}

// GetEventsForTimeRange gets events for a specific time range. If calendarName is empty, fetch from all calendars
func (cs *CalendarService) GetEventsForTimeRange(ctx context.Context, start, end time.Time, calendarName string) ([]*calendar.Event, error) {
	calendarNames := []string{}
	if calendarName == "" {
		// If calendarName is empty, fetch events from all calendars
		for _, cal := range cs.calendarConfig.Calendars {
			calendarNames = append(calendarNames, cal.Name)
		}
	} else {
		// Otherwise, just use the specified calendar
		calendarNames = append(calendarNames, calendarName)
	}

	allEvents := []*calendar.Event{}
	for _, calName := range calendarNames {
		// Fetch events for each calendar
		calendarID := cs.getCalendarID(calName)

		call := cs.service.Events.List(calendarID).
			TimeMin(start.Format(time.RFC3339)).
			TimeMax(end.Format(time.RFC3339)).
			SingleEvents(true).
			OrderBy("startTime")

		events, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to get events: %w", err)
		}

		allEvents = append(allEvents, events.Items...)
	}

	// Sort events so that start time is descending
	var j int
	for i := 1; i < len(allEvents); i++ {
		event := allEvents[i]

		// Parse pivot event time
		t0, err := time.Parse(time.RFC3339, event.Start.DateTime)
		if err != nil {
			continue
		}

		for j = i - 1; j >= 0; j-- {
			// Parse comparison event time
			t1, err := time.Parse(time.RFC3339, allEvents[j].Start.DateTime)
			if err != nil || t0.After(t1) {
				break
			}
			allEvents[j+1] = allEvents[j]
		}

		allEvents[j+1] = event
	}

	return allEvents, nil
}

// GetTodayEvents gets events for today. If calendarName is empty, fetch all calendars
func (cs *CalendarService) GetTodayEvents(ctx context.Context, calendarName string) ([]*calendar.Event, error) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.Add(24 * time.Hour)

	calendarNames := []string{}
	if calendarName == "" {
		// If calendarName is empty, fetch events from all calendars
		for _, cal := range cs.calendarConfig.Calendars {
			calendarNames = append(calendarNames, cal.Name)
		}
	} else {
		// Otherwise, just use the specified calendar
		calendarNames = append(calendarNames, calendarName)
	}

	allEvents := []*calendar.Event{}
	for _, calName := range calendarNames {
		events, err := cs.GetEventsForTimeRange(ctx, start, end, calName)
		if err != nil {
			return nil, err
		}
		allEvents = append(allEvents, events...)
	}

	return allEvents, nil
}

// GetWeekEvents gets events for this week. If calendarName is empty, fetch all calendars
func (cs *CalendarService) GetWeekEvents(ctx context.Context, calendarName string) ([]*calendar.Event, error) {
	now := time.Now()
	weekday := int(now.Weekday())
	start := now.AddDate(0, 0, -weekday) // Start of week (Sunday)
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	end := start.Add(7 * 24 * time.Hour)

	calendarNames := []string{}
	if calendarName == "" {
		// If calendarName is empty, fetch events from all calendars
		for _, cal := range cs.calendarConfig.Calendars {
			calendarNames = append(calendarNames, cal.Name)
		}
	} else {
		// Otherwise, just use the specified calendar
		calendarNames = append(calendarNames, calendarName)
	}

	allEvents := []*calendar.Event{}
	for _, calName := range calendarNames {
		events, err := cs.GetEventsForTimeRange(ctx, start, end, calName)
		if err != nil {
			return nil, err
		}
		allEvents = append(allEvents, events...)
	}

	return allEvents, nil
}

// CreateEvent creates a new calendar event
func (cs *CalendarService) CreateEvent(ctx context.Context, title, description string, start, end time.Time, calendarName string) (*calendar.Event, error) {
	calendarID := cs.getCalendarID(calendarName)
	if calendarID == "" {
		return nil, fmt.Errorf("invalid calendar name: %s", calendarName)
	}

	event := &calendar.Event{
		Summary:     title,
		Description: description,
		Start: &calendar.EventDateTime{
			DateTime: start.Format(time.RFC3339),
			TimeZone: start.Location().String(),
		},
		End: &calendar.EventDateTime{
			DateTime: end.Format(time.RFC3339),
			TimeZone: end.Location().String(),
		},
	}

	createdEvent, err := cs.service.Events.Insert(calendarID, event).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	return createdEvent, nil
}

// UpdateEvent updates an existing calendar event
func (cs *CalendarService) UpdateEvent(ctx context.Context, eventID string, title, description string, start, end time.Time, calendarName string) (*calendar.Event, error) {
	calendarID := cs.getCalendarID(calendarName)
	if calendarID == "" {
		return nil, fmt.Errorf("invalid calendar name: %s", calendarName)
	}

	// First get the existing event
	existingEvent, err := cs.service.Events.Get(calendarID, eventID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get existing event: %w", err)
	}

	// Update the fields
	if title != "" {
		existingEvent.Summary = title
	}
	if description != "" {
		existingEvent.Description = description
	}
	if !start.IsZero() {
		existingEvent.Start = &calendar.EventDateTime{
			DateTime: start.Format(time.RFC3339),
			TimeZone: start.Location().String(),
		}
	}
	if !end.IsZero() {
		existingEvent.End = &calendar.EventDateTime{
			DateTime: end.Format(time.RFC3339),
			TimeZone: end.Location().String(),
		}
	}

	updatedEvent, err := cs.service.Events.Update(calendarID, eventID, existingEvent).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	return updatedEvent, nil
}

// DeleteEvent deletes a calendar event
func (cs *CalendarService) DeleteEvent(ctx context.Context, eventID string, calendarName string) error {
	calendarID := cs.getCalendarID(calendarName)
	if calendarID == "" {
		return fmt.Errorf("invalid calendar name: %s", calendarName)
	}

	err := cs.service.Events.Delete(calendarID, eventID).Do()
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	return nil
}

// Helper function to map calendar name to IDs
func (cs *CalendarService) getCalendarID(calendarName string) string {
	// Map calendar name to actual calendar IDs
	if calendarName != "" {
		for _, cal := range cs.calendarConfig.Calendars {
			if strings.EqualFold(cal.Name, calendarName) {
				return cal.ID
			}
		}
	}
	return ""
}

// saveToken saves the OAuth2 token to a file
func saveToken(path string, token *oauth2.Token) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to save token: %w", err)
	}
	defer f.Close()
	
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	return encoder.Encode(token)
}
