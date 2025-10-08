package schedule

import (
	"context"
	"log"
	"testing"

	"github.com/ethanbaker/assistant/internal/stores/memory"
	"github.com/ethanbaker/assistant/internal/stores/session"
	"github.com/ethanbaker/assistant/pkg/utils"
)

var sa *ScheduleAgent

// Initialize schedule agent for testing
func getScheduleAgent() *ScheduleAgent {
	if sa != nil {
		return sa
	}

	// Create config
	envFile := ".env.test"
	cfg := utils.NewConfigFromEnv(envFile)

	// Create mock stores
	memoryStore := &memory.Store{}           // Mock implementation
	sessionStore := &session.InMemoryStore{} // Mock implementation

	var err error
	sa, err = NewScheduleAgent(memoryStore, sessionStore, cfg)
	if err != nil {
		log.Fatalf("Failed to create schedule agent: %v", err)
	}

	return sa
}

func TestHandleSearchEvents(t *testing.T) {
	sa := getScheduleAgent()
	ctx := context.Background()

	tests := []struct {
		name      string
		arguments string
		wantError bool
	}{
		{
			name:      "Valid search with calendar name",
			arguments: `{"query": "meeting", "calendar_name": "primary"}`,
			wantError: false,
		},
		{
			name:      "Valid search without calendar name",
			arguments: `{"query": "meeting"}`,
			wantError: false,
		},
		{
			name:      "Empty query",
			arguments: `{"query": "", "calendar_name": "primary"}`,
			wantError: true,
		},
		{
			name:      "Missing query field",
			arguments: `{"calendar_name": "primary"}`,
			wantError: true,
		},
		{
			name:      "Invalid JSON",
			arguments: `{"query": invalid}`,
			wantError: true,
		},
		{
			name:      "Invalid calendar name",
			arguments: `{"query": "meeting", "calendar_name": "invalid_calendar"}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sa.handleSearchEvents(ctx, tt.arguments)

			if tt.wantError && err == nil {
				t.Errorf("handleSearchEvents() expected error for test %s but got none", tt.name)
			}
			if !tt.wantError && err != nil {
				t.Errorf("handleSearchEvents() unexpected error for test %s: %v", tt.name, err)
			}

			// Note: In actual implementation, you would mock the underlying
			// calendarService.SearchEvents method to return expected results
			_ = result
		})
	}
}

func TestHandleGetTodayEvents(t *testing.T) {
	sa := getScheduleAgent()
	ctx := context.Background()

	tests := []struct {
		name      string
		arguments string
		wantError bool
	}{
		{
			name:      "Valid request with calendar name",
			arguments: `{"calendar_name": "primary"}`,
			wantError: false,
		},
		{
			name:      "Empty arguments",
			arguments: `{}`,
			wantError: false, // calendar_name is required in schema
		},
		{
			name:      "Invalid JSON",
			arguments: `{"calendar_name": invalid}`,
			wantError: true,
		},
		{
			name:      "Invalid calendar name",
			arguments: `{"calendar_name": "invalid_calendar"}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sa.handleGetTodayEvents(ctx, tt.arguments)

			if tt.wantError && err == nil {
				t.Errorf("handleGetTodayEvents() expected error for test %s but got none", tt.name)
			}
			if !tt.wantError && err != nil {
				t.Errorf("handleGetTodayEvents() unexpected error for test %s: %v", tt.name, err)
			}

			// Note: In actual implementation, you would mock the underlying
			// calendarService.GetTodayEvents method to return expected results
			_ = result
		})
	}
}

func TestHandleGetWeekEvents(t *testing.T) {
	sa := getScheduleAgent()
	ctx := context.Background()

	tests := []struct {
		name      string
		arguments string
		wantError bool
	}{
		{
			name:      "Valid request with calendar name",
			arguments: `{"calendar_name": "primary"}`,
			wantError: false,
		},
		{
			name:      "Empty arguments",
			arguments: `{}`,
			wantError: false,
		},
		{
			name:      "Invalid JSON",
			arguments: `{"calendar_name": invalid}`,
			wantError: true,
		},
		{
			name:      "Invalid calendar name",
			arguments: `{"calendar_name": "invalid_calendar"}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sa.handleGetWeekEvents(ctx, tt.arguments)

			if tt.wantError && err == nil {
				t.Errorf("handleGetWeekEvents() expected error for test %s but got none", tt.name)
			}
			if !tt.wantError && err != nil {
				t.Errorf("handleGetWeekEvents() unexpected error for test %s: %v", tt.name, err)
			}

			// Note: In actual implementation, you would mock the underlying
			// calendarService.GetWeekEvents method to return expected results
			_ = result
		})
	}
}

func TestHandleGetMonthEvents(t *testing.T) {
	sa := getScheduleAgent()
	ctx := context.Background()

	tests := []struct {
		name      string
		arguments string
		wantError bool
	}{
		{
			name:      "Valid request with calendar name",
			arguments: `{"calendar_name": "primary"}`,
			wantError: false,
		},
		{
			name:      "Empty arguments",
			arguments: `{}`,
			wantError: false,
		},
		{
			name:      "Invalid JSON",
			arguments: `{"calendar_name": invalid}`,
			wantError: true,
		},
		{
			name:      "Invalid calendar name",
			arguments: `{"calendar_name": "invalid_calendar"}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sa.handleGetMonthEvents(ctx, tt.arguments)

			if tt.wantError && err == nil {
				t.Errorf("handleGetMonthEvents() expected error for test %s but got none", tt.name)
			}
			if !tt.wantError && err != nil {
				t.Errorf("handleGetMonthEvents() unexpected error for test %s: %v", tt.name, err)
			}

			// Note: In actual implementation, you would mock the underlying
			// calendarService.GetEventsForTimeRange method to return expected results
			_ = result
		})
	}
}

func TestHandleGetSpecificDayEvents(t *testing.T) {
	sa := getScheduleAgent()
	ctx := context.Background()

	tests := []struct {
		name      string
		arguments string
		wantError bool
	}{
		{
			name:      "Valid request with date and calendar",
			arguments: `{"date": "2025-10-05", "calendar_name": "primary"}`,
			wantError: false,
		},
		{
			name:      "Valid date different format",
			arguments: `{"date": "2025-12-31", "calendar_name": "primary"}`,
			wantError: false,
		},
		{
			name:      "No calendar name (fetch all calendars)",
			arguments: `{"date": "2025-10-05"}`,
			wantError: false,
		},
		{
			name:      "Missing date field",
			arguments: `{"calendar_name": "primary"}`,
			wantError: true,
		},
		{
			name:      "Empty date",
			arguments: `{"date": "", "calendar_name": "primary"}`,
			wantError: true,
		},
		{
			name:      "Invalid date format",
			arguments: `{"date": "10/05/2025", "calendar_name": "primary"}`,
			wantError: true,
		},
		{
			name:      "Invalid date format 2",
			arguments: `{"date": "2025-13-01", "calendar_name": "primary"}`,
			wantError: true,
		},
		{
			name:      "Invalid date format 3",
			arguments: `{"date": "invalid-date", "calendar_name": "primary"}`,
			wantError: true,
		},
		{
			name:      "Invalid calendar name",
			arguments: `{"date": "2025-10-05", "calendar_name": "invalid_calendar"}`,
			wantError: true,
		},
		{
			name:      "Invalid JSON",
			arguments: `{"date": invalid}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sa.handleGetSpecificDayEvents(ctx, tt.arguments)

			if tt.wantError && err == nil {
				t.Errorf("handleGetSpecificDayEvents() expected error for test %s but got none", tt.name)
			}
			if !tt.wantError && err != nil {
				t.Errorf("handleGetSpecificDayEvents() unexpected error for test %s: %v", tt.name, err)
			}

			// Note: In actual implementation, you would mock the underlying
			// calendarService.GetEventsForTimeRange method to return expected results
			_ = result
		})
	}
}

func TestHandleCreateEvent(t *testing.T) {
	sa := getScheduleAgent()
	ctx := context.Background()

	tests := []struct {
		name      string
		arguments string
		wantError bool
	}{
		{
			name: "Valid event creation",
			arguments: `{
				"title": "Test Meeting",
				"description": "Test meeting description",
				"start_time": "2025-10-05T10:00:00Z",
				"end_time": "2025-10-05T11:00:00Z",
				"calendar_name": "primary"
			}`,
			wantError: false,
		},
		{
			name: "Valid event with timezone",
			arguments: `{
				"title": "Another Meeting",
				"description": "Another test meeting",
				"start_time": "2025-10-05T14:00:00-07:00",
				"end_time": "2025-10-05T15:30:00-07:00",
				"calendar_name": "primary"
			}`,
			wantError: false,
		},
		{
			name:      "Missing title",
			arguments: `{"description": "Test", "start_time": "2025-10-05T10:00:00Z", "end_time": "2025-10-05T11:00:00Z", "calendar_name": "primary"}`,
			wantError: true,
		},
		{
			name:      "Empty title",
			arguments: `{"title": "", "description": "Test", "start_time": "2025-10-05T10:00:00Z", "end_time": "2025-10-05T11:00:00Z", "calendar_name": "primary"}`,
			wantError: true,
		},
		{
			name:      "Missing start_time",
			arguments: `{"title": "Test", "description": "Test", "end_time": "2025-10-05T11:00:00Z", "calendar_name": "primary"}`,
			wantError: true,
		},
		{
			name:      "Empty start_time",
			arguments: `{"title": "Test", "description": "Test", "start_time": "", "end_time": "2025-10-05T11:00:00Z", "calendar_name": "primary"}`,
			wantError: true,
		},
		{
			name:      "Missing end_time",
			arguments: `{"title": "Test", "description": "Test", "start_time": "2025-10-05T10:00:00Z", "calendar_name": "primary"}`,
			wantError: true,
		},
		{
			name:      "Empty end_time",
			arguments: `{"title": "Test", "description": "Test", "start_time": "2025-10-05T10:00:00Z", "end_time": "", "calendar_name": "primary"}`,
			wantError: true,
		},
		{
			name:      "Missing calendar_name",
			arguments: `{"title": "Test", "description": "Test", "start_time": "2025-10-05T10:00:00Z", "end_time": "2025-10-05T11:00:00Z"}`,
			wantError: true,
		},
		{
			name:      "Empty calendar_name",
			arguments: `{"title": "Test", "description": "Test", "start_time": "2025-10-05T10:00:00Z", "end_time": "2025-10-05T11:00:00Z", "calendar_name": ""}`,
			wantError: true,
		},
		{
			name:      "Invalid calendar_name",
			arguments: `{"title": "Test", "description": "Test", "start_time": "2025-10-05T10:00:00Z", "end_time": "2025-10-05T11:00:00Z", "calendar_name": "invalid_calendar"}`,
			wantError: true,
		},
		{
			name:      "Invalid start_time format",
			arguments: `{"title": "Test", "description": "Test", "start_time": "2025-10-05 10:00:00", "end_time": "2025-10-05T11:00:00Z", "calendar_name": "primary"}`,
			wantError: true,
		},
		{
			name:      "Invalid end_time format",
			arguments: `{"title": "Test", "description": "Test", "start_time": "2025-10-05T10:00:00Z", "end_time": "2025-10-05 11:00:00", "calendar_name": "primary"}`,
			wantError: true,
		},
		{
			name: "End time before start time",
			arguments: `{
				"title": "Test",
				"description": "Test",
				"start_time": "2025-10-05T11:00:00Z",
				"end_time": "2025-10-05T10:00:00Z",
				"calendar_name": "primary"
			}`,
			wantError: true,
		},
		{
			name: "End time equal to start time",
			arguments: `{
				"title": "Test",
				"description": "Test",
				"start_time": "2025-10-05T10:00:00Z",
				"end_time": "2025-10-05T10:00:00Z",
				"calendar_name": "primary"
			}`,
			wantError: true,
		},
		{
			name:      "Invalid JSON",
			arguments: `{"title": invalid}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sa.handleCreateEvent(ctx, tt.arguments)

			if tt.wantError && err == nil {
				t.Errorf("handleCreateEvent() expected error for test %s but got none", tt.name)
			}
			if !tt.wantError && err != nil {
				t.Errorf("handleCreateEvent() unexpected error for test %s: %v", tt.name, err)
			}

			// Note: In actual implementation, you would mock the underlying
			// calendarService.CreateEvent method to return expected results
			_ = result
		})
	}
}

func TestHandleUpdateEvent(t *testing.T) {
	sa := getScheduleAgent()
	ctx := context.Background()

	tests := []struct {
		name      string
		arguments string
		wantError bool
	}{
		{
			name: "Valid partial update - title only",
			arguments: `{
				"event_id": "test-event-123",
				"calendar_name": "primary",
				"title": "Updated Meeting"
			}`,
			wantError: false,
		},
		{
			name: "Valid full update",
			arguments: `{
				"event_id": "test-event-123",
				"calendar_name": "primary",
				"title": "Updated Meeting",
				"description": "Updated description",
				"start_time": "2025-10-05T10:00:00Z",
				"end_time": "2025-10-05T11:00:00Z"
			}`,
			wantError: false,
		},
		{
			name: "Valid update with timezone",
			arguments: `{
				"event_id": "test-event-123",
				"calendar_name": "primary",
				"start_time": "2025-10-05T14:00:00-07:00",
				"end_time": "2025-10-05T15:30:00-07:00"
			}`,
			wantError: false,
		},
		{
			name:      "Missing event_id",
			arguments: `{"calendar_name": "primary", "title": "Updated Meeting"}`,
			wantError: true,
		},
		{
			name:      "Empty event_id",
			arguments: `{"event_id": "", "calendar_name": "primary", "title": "Updated Meeting"}`,
			wantError: true,
		},
		{
			name:      "Missing calendar_name",
			arguments: `{"event_id": "test-event-123", "title": "Updated Meeting"}`,
			wantError: true,
		},
		{
			name:      "Empty calendar_name",
			arguments: `{"event_id": "test-event-123", "calendar_name": "", "title": "Updated Meeting"}`,
			wantError: true,
		},
		{
			name:      "Invalid calendar_name",
			arguments: `{"event_id": "test-event-123", "calendar_name": "invalid_calendar", "title": "Updated Meeting"}`,
			wantError: true,
		},
		{
			name:      "Invalid start_time format",
			arguments: `{"event_id": "test-event-123", "calendar_name": "primary", "start_time": "2025-10-05 10:00:00"}`,
			wantError: true,
		},
		{
			name:      "Invalid end_time format",
			arguments: `{"event_id": "test-event-123", "calendar_name": "primary", "end_time": "2025-10-05 11:00:00"}`,
			wantError: true,
		},
		{
			name: "End time before start time",
			arguments: `{
				"event_id": "test-event-123",
				"calendar_name": "primary",
				"start_time": "2025-10-05T11:00:00Z",
				"end_time": "2025-10-05T10:00:00Z"
			}`,
			wantError: true,
		},
		{
			name: "End time equal to start time",
			arguments: `{
				"event_id": "test-event-123",
				"calendar_name": "primary",
				"start_time": "2025-10-05T10:00:00Z",
				"end_time": "2025-10-05T10:00:00Z"
			}`,
			wantError: true,
		},
		{
			name:      "Invalid JSON",
			arguments: `{"event_id": invalid}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sa.handleUpdateEvent(ctx, tt.arguments)

			if tt.wantError && err == nil {
				t.Errorf("handleUpdateEvent() expected error for test %s but got none", tt.name)
			}
			if !tt.wantError && err != nil {
				t.Errorf("handleUpdateEvent() unexpected error for test %s: %v", tt.name, err)
			}

			// Note: In actual implementation, you would mock the underlying
			// calendarService.UpdateEvent method to return expected results
			_ = result
		})
	}
}

func TestHandleDeleteEvent(t *testing.T) {
	sa := getScheduleAgent()
	ctx := context.Background()

	tests := []struct {
		name      string
		arguments string
		wantError bool
	}{
		{
			name:      "Valid deletion",
			arguments: `{"event_id": "test-event-123", "calendar_name": "primary"}`,
			wantError: false,
		},
		{
			name:      "Another valid deletion",
			arguments: `{"event_id": "another-event-456", "calendar_name": "primary"}`,
			wantError: false,
		},
		{
			name:      "Missing event_id",
			arguments: `{"calendar_name": "primary"}`,
			wantError: true,
		},
		{
			name:      "Empty event_id",
			arguments: `{"event_id": "", "calendar_name": "primary"}`,
			wantError: true,
		},
		{
			name:      "Missing calendar_name",
			arguments: `{"event_id": "test-event-123"}`,
			wantError: true,
		},
		{
			name:      "Empty calendar_name",
			arguments: `{"event_id": "test-event-123", "calendar_name": ""}`,
			wantError: true,
		},
		{
			name:      "Invalid calendar_name",
			arguments: `{"event_id": "test-event-123", "calendar_name": "invalid_calendar"}`,
			wantError: true,
		},
		{
			name:      "Invalid JSON",
			arguments: `{"event_id": invalid}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sa.handleDeleteEvent(ctx, tt.arguments)

			if tt.wantError && err == nil {
				t.Errorf("handleDeleteEvent() expected error for test %s but got none", tt.name)
			}
			if !tt.wantError && err != nil {
				t.Errorf("handleDeleteEvent() unexpected error for test %s: %v", tt.name, err)
			}

			// Verify response structure when no error is expected
			if !tt.wantError && err == nil {
				if result == nil {
					t.Errorf("handleDeleteEvent() returned nil result for test %s", tt.name)
				}

				// Check if result has expected structure
				if response, ok := result.(map[string]any); ok {
					if success, exists := response["success"]; !exists || success != true {
						t.Errorf("handleDeleteEvent() missing or invalid success field for test %s", tt.name)
					}
					if message, exists := response["message"]; !exists || message != "Event deleted successfully" {
						t.Errorf("handleDeleteEvent() missing or invalid message field for test %s", tt.name)
					}
					if eventID, exists := response["event_id"]; !exists || eventID == "" {
						t.Errorf("handleDeleteEvent() missing or invalid event_id field for test %s", tt.name)
					}
					if calendarName, exists := response["calendar_name"]; !exists || calendarName == "" {
						t.Errorf("handleDeleteEvent() missing or invalid calendar_name field for test %s", tt.name)
					}
				} else {
					t.Errorf("handleDeleteEvent() returned unexpected result type for test %s", tt.name)
				}
			}

			// Note: In actual implementation, you would mock the underlying
			// calendarService.DeleteEvent method to return expected results
			_ = result
		})
	}
}
