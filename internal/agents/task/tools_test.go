package task

import (
	"context"
	"strings"
	"testing"
)

func TestNormalizeArguments(t *testing.T) {
	if got := normalizeArguments(""); got != "{}" {
		t.Fatalf("expected empty arguments to normalize to {}, got %q", got)
	}

	if got := normalizeArguments("   "); got != "{}" {
		t.Fatalf("expected whitespace arguments to normalize to {}, got %q", got)
	}

	if got := normalizeArguments(`{"x":1}`); got != `{"x":1}` {
		t.Fatalf("expected non-empty arguments to remain unchanged, got %q", got)
	}
}

func TestHandleFetchTasksValidation(t *testing.T) {
	ta := &TaskAgent{}
	ctx := context.Background()

	_, err := ta.handleFetchTasks(ctx, `{"priority":"not-valid"}`)
	if err == nil {
		t.Fatalf("expected invalid priority to return an error")
	}
	if !strings.Contains(err.Error(), "invalid priority value") {
		t.Fatalf("expected invalid priority error, got %v", err)
	}

	_, err = ta.handleFetchTasks(ctx, `{"due_date":"2026/03/07"}`)
	if err == nil {
		t.Fatalf("expected invalid due_date to return an error")
	}
	if !strings.Contains(err.Error(), "invalid due_date format") {
		t.Fatalf("expected invalid due_date error, got %v", err)
	}
}

func TestHandleCreateNewTaskValidation(t *testing.T) {
	ta := &TaskAgent{}
	ctx := context.Background()

	_, err := ta.handleCreateNewTask(ctx, `{}`)
	if err == nil {
		t.Fatalf("expected missing title to return an error")
	}
	if !strings.Contains(err.Error(), "title is required") {
		t.Fatalf("expected title required error, got %v", err)
	}
}
