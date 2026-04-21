package schedule

import (
	"context"
	"strings"
	"testing"
)

func testScheduleAgent() *ScheduleAgent {
	return &ScheduleAgent{}
}

func TestNormalizeArguments(t *testing.T) {
	if got := normalizeArguments(""); got != "{}" {
		t.Fatalf("expected empty arguments to normalize to {}, got %q", got)
	}

	if got := normalizeArguments("  "); got != "{}" {
		t.Fatalf("expected whitespace arguments to normalize to {}, got %q", got)
	}

	if got := normalizeArguments(`{"x":1}`); got != `{"x":1}` {
		t.Fatalf("expected non-empty arguments to remain unchanged, got %q", got)
	}
}

func TestHandleSearchEventsValidation(t *testing.T) {
	sa := testScheduleAgent()
	ctx := context.Background()

	_, err := sa.handleSearchEvents(ctx, `{}`)
	if err == nil {
		t.Fatalf("expected missing query to return an error")
	}
	if !strings.Contains(err.Error(), "query is required") {
		t.Fatalf("expected query required error, got %v", err)
	}
}

func TestHandleGetSpecificDayEventsValidation(t *testing.T) {
	sa := testScheduleAgent()
	ctx := context.Background()

	_, err := sa.handleGetSpecificDayEvents(ctx, `{}`)
	if err == nil {
		t.Fatalf("expected missing date to return an error")
	}
	if !strings.Contains(err.Error(), "date is required") {
		t.Fatalf("expected date required error, got %v", err)
	}

	_, err = sa.handleGetSpecificDayEvents(ctx, `{"date":"03/07/2026"}`)
	if err == nil {
		t.Fatalf("expected invalid date format to return an error")
	}
	if !strings.Contains(err.Error(), "invalid date format") {
		t.Fatalf("expected invalid date format error, got %v", err)
	}
}

func TestHandleCreateEventValidation(t *testing.T) {
	sa := testScheduleAgent()
	ctx := context.Background()

	_, err := sa.handleCreateEvent(ctx, `{}`)
	if err == nil {
		t.Fatalf("expected missing title to return an error")
	}
	if !strings.Contains(err.Error(), "title is required") {
		t.Fatalf("expected title required error, got %v", err)
	}
}

func TestHandleUpdateEventValidation(t *testing.T) {
	sa := testScheduleAgent()
	ctx := context.Background()

	_, err := sa.handleUpdateEvent(ctx, `{}`)
	if err == nil {
		t.Fatalf("expected missing event_id to return an error")
	}
	if !strings.Contains(err.Error(), "event_id is required") {
		t.Fatalf("expected event_id required error, got %v", err)
	}
}

func TestHandleDeleteEventValidation(t *testing.T) {
	sa := testScheduleAgent()
	ctx := context.Background()

	_, err := sa.handleDeleteEvent(ctx, `{}`)
	if err == nil {
		t.Fatalf("expected missing event_id to return an error")
	}
	if !strings.Contains(err.Error(), "event_id is required") {
		t.Fatalf("expected event_id required error, got %v", err)
	}
}
