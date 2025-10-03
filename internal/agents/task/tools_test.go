package task

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"testing"
	"time"

	notionapi "github.com/dstotijn/go-notion"
	"github.com/ethanbaker/assistant/internal/stores/memory"
	"github.com/ethanbaker/assistant/internal/stores/session"
	"github.com/ethanbaker/assistant/pkg/utils"
)

var ta *TaskAgent

// Initialize task agent
func getTaskAgent() *TaskAgent {
	if ta != nil {
		return ta
	}

	// Create config
	envFile := ".env.test"
	cfg := utils.NewConfigFromEnv(envFile)

	// Get notion token
	token := cfg.Get("NOTION_API_TOKEN")
	if token == "" {
		log.Fatal("NOTION_API_TOKEN not set in environment")
	}

	// Create mock stores and config
	memoryStore := &memory.Store{}           // Mock implementation
	sessionStore := &session.InMemoryStore{} // Mock implementation

	notion := notionapi.NewClient(token, notionapi.WithHTTPClient(&http.Client{
		Timeout: 20 * time.Second,
	}))

	ta = &TaskAgent{
		config:       cfg,
		memoryStore:  memoryStore,
		sessionStore: sessionStore,
		notionClient: notion,
		basePrompt:   "test prompt",
	}

	return ta
}

func TestHandleFetchTasks(t *testing.T) {
	ta := getTaskAgent()
	ctx := context.Background()

	tests := []struct {
		name      string
		arguments string
		wantError bool
	}{
		{
			name:      "Valid empty arguments",
			arguments: "{}",
			wantError: false,
		},
		{
			name: "Valid arguments with all filters",
			arguments: `{
				"complete": true,
				"priority": "High (3)",
				"effort": "Medium (2)",
				"due_date": "2025-10-03"
			}`,
			wantError: false,
		},
		{
			name: "Valid arguments with partial filters",
			arguments: `{
				"complete": false,
				"priority": "Low (1)"
			}`,
			wantError: false,
		},
		{
			name:      "Invalid JSON",
			arguments: `{"invalid": json}`,
			wantError: true,
		},
		{
			name: "Invalid priority value",
			arguments: `{
				"priority": "Invalid Priority"
			}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ta.handleFetchTasks(ctx, tt.arguments)

			if tt.wantError && err == nil {
				t.Errorf("handleFetchTasks() expected error for test %s but got none", tt.name)
			}
			if !tt.wantError && err != nil {
				t.Errorf("handleFetchTasks() unexpected error for test %s: %v", tt.name, err)
			}

			// Note: In actual implementation, you would mock the underlying
			// queryTasks method to return expected results
			_ = result
		})
	}
}

func TestHandleGetTaskDetails(t *testing.T) {
	ta := getTaskAgent()
	ctx := context.Background()

	tests := []struct {
		name      string
		arguments string
		wantError bool
	}{
		{
			name:      "Valid task ID",
			arguments: `{"task_id": "814396650eaf4b67b169a05815bed9f6"}`,
			wantError: false,
		},
		{
			name:      "Empty task ID",
			arguments: `{"task_id": ""}`,
			wantError: true,
		},
		{
			name:      "Missing task_id field",
			arguments: `{}`,
			wantError: true,
		},
		{
			name:      "Invalid JSON",
			arguments: `{"task_id": invalid}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ta.handleGetTaskDetails(ctx, tt.arguments)

			if tt.wantError && err == nil {
				t.Errorf("handleGetTaskDetails() expected error for test %s but got none", tt.name)
			}
			if !tt.wantError && err != nil {
				t.Errorf("handleGetTaskDetails() unexpected error for test %s: %v", tt.name, err)
			}

			// Note: In actual implementation, you would mock the underlying
			// getTaskDetails method to return expected results
			_ = result
		})
	}
}

func TestHandleGetUpcomingTasks(t *testing.T) {
	ta := getTaskAgent()
	ctx := context.Background()

	tests := []struct {
		name      string
		arguments string
		wantError bool
	}{
		{
			name:      "Empty arguments",
			arguments: "",
			wantError: false,
		},
		{
			name:      "JSON arguments (ignored)",
			arguments: `{"ignored": "value"}`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ta.handleGetUpcomingTasks(ctx, tt.arguments)

			if tt.wantError && err == nil {
				t.Errorf("handleGetUpcomingTasks() expected error for test %s but got none", tt.name)
			}
			if !tt.wantError && err != nil {
				t.Errorf("handleGetUpcomingTasks() unexpected error for test %s: %v", tt.name, err)
			}

			// Note: In actual implementation, you would mock the underlying
			// queryTasks method to return expected results
			_ = result
		})
	}
}

func TestHandleGetRecurringTasks(t *testing.T) {
	ta := getTaskAgent()
	ctx := context.Background()

	tests := []struct {
		name      string
		arguments string
		wantError bool
	}{
		{
			name:      "Empty arguments",
			arguments: "",
			wantError: false,
		},
		{
			name:      "JSON arguments (ignored)",
			arguments: `{"ignored": "value"}`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ta.handleGetRecurringTasks(ctx, tt.arguments)

			if tt.wantError && err == nil {
				t.Errorf("handleGetRecurringTasks() expected error for test %s but got none", tt.name)
			}
			if !tt.wantError && err != nil {
				t.Errorf("handleGetRecurringTasks() unexpected error for test %s: %v", tt.name, err)
			}

			// Note: In actual implementation, you would mock the underlying
			// queryRecurringTasks method to return expected results
			_ = result
		})
	}
}

func TestHandleCreateNewTask(t *testing.T) {
	ta := getTaskAgent()
	ctx := context.Background()

	tests := []struct {
		name      string
		arguments string
		wantError bool
	}{
		{
			name:      "Valid minimal task",
			arguments: `{"title": "Test Task"}`,
			wantError: false,
		},
		{
			name: "Valid task with all fields",
			arguments: `{
				"title": "Complete Task",
				"priority": "High (3)",
				"effort": "Medium (2)",
				"due_date": "2025-10-03",
				"project": "Test Project"
			}`,
			wantError: false,
		},
		{
			name:      "Missing title",
			arguments: `{}`,
			wantError: true,
		},
		{
			name:      "Empty title",
			arguments: `{"title": ""}`,
			wantError: true,
		},
		{
			name:      "Invalid JSON",
			arguments: `{"title": invalid}`,
			wantError: true,
		},
		{
			name: "Invalid priority",
			arguments: `{
				"title": "Test Task",
				"priority": "Invalid Priority"
			}`,
			wantError: true,
		},
		{
			name: "Invalid effort",
			arguments: `{
				"title": "Test Task",
				"effort": "Invalid Effort"
			}`,
			wantError: true,
		},
		{
			name: "Invalid date format",
			arguments: `{
				"title": "Test Task",
				"due_date": "invalid-date"
			}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ta.handleCreateNewTask(ctx, tt.arguments)

			if tt.wantError && err == nil {
				t.Errorf("handleCreateNewTask() expected error for test %s but got none", tt.name)
			}
			if !tt.wantError && err != nil {
				t.Errorf("handleCreateNewTask() unexpected error for test %s: %v", tt.name, err)
			}

			// Note: In actual implementation, you would mock the underlying
			// createTask method to return expected results
			_ = result
		})
	}
}

func TestHandleUpdateTask(t *testing.T) {
	ta := getTaskAgent()
	ctx := context.Background()

	tests := []struct {
		name      string
		arguments string
		wantError bool
	}{
		{
			name: "Valid partial update",
			arguments: `{
				"task_id": "814396650eaf4b67b169a05815bed9f6",
				"title": "Updated Task 1"
			}`,
			wantError: false,
		},
		{
			name: "Valid update with all fields",
			arguments: `{
				"task_id": "814396650eaf4b67b169a05815bed9f6",
				"title": "Updated Task 2",
				"priority": "High (3)",
				"effort": "Medium (2)",
				"due_date": "2025-10-03",
				"project": "Updated Project"
			}`,
			wantError: false,
		},
		{
			name:      "Missing task_id",
			arguments: `{"title": "Updated Task"}`,
			wantError: true,
		},
		{
			name:      "Empty task_id",
			arguments: `{"task_id": ""}`,
			wantError: true,
		},
		{
			name:      "Invalid JSON",
			arguments: `{"task_id": invalid}`,
			wantError: true,
		},
		{
			name: "Invalid priority",
			arguments: `{
				"task_id": "test-task-id-123",
				"priority": "Invalid Priority"
			}`,
			wantError: true,
		},
		{
			name: "Invalid effort",
			arguments: `{
				"task_id": "test-task-id-123",
				"effort": "Invalid Effort"
			}`,
			wantError: true,
		},
		{
			name: "Invalid date format",
			arguments: `{
				"task_id": "test-task-id-123",
				"due_date": "invalid-date"
			}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ta.handleUpdateTask(ctx, tt.arguments)

			if tt.wantError && err == nil {
				t.Errorf("handleUpdateTask() expected error for test %s but got none", tt.name)
			}
			if !tt.wantError && err != nil {
				t.Errorf("handleUpdateTask() unexpected error for test %s: %v", tt.name, err)
			}

			// Note: In actual implementation, you would mock the underlying
			// updateTask method to return expected results
			_ = result
		})
	}
}

func TestHandleCompleteTask(t *testing.T) {
	ta := getTaskAgent()
	ctx := context.Background()

	tests := []struct {
		name      string
		arguments string
		wantError bool
	}{
		{
			name:      "Valid task ID",
			arguments: `{"task_id": "814396650eaf4b67b169a05815bed9f6"}`,
			wantError: false,
		},
		{
			name:      "Empty task ID",
			arguments: `{"task_id": ""}`,
			wantError: true,
		},
		{
			name:      "Missing task_id field",
			arguments: `{}`,
			wantError: true,
		},
		{
			name:      "Invalid JSON",
			arguments: `{"task_id": invalid}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ta.handleCompleteTask(ctx, tt.arguments)

			if tt.wantError && err == nil {
				t.Errorf("handleCompleteTask() expected error for test %s but got none", tt.name)
			}
			if !tt.wantError && err != nil {
				t.Errorf("handleCompleteTask() unexpected error for test %s: %v", tt.name, err)
			}

			// Note: In actual implementation, you would mock the underlying
			// completeTask method to return expected results
			_ = result
		})
	}
}

func TestHandleHighlightBlockers(t *testing.T) {
	ta := getTaskAgent()
	ctx := context.Background()

	tests := []struct {
		name      string
		arguments string
		wantError bool
	}{
		{
			name:      "Empty arguments",
			arguments: "",
			wantError: false,
		},
		{
			name:      "JSON arguments (ignored)",
			arguments: `{"ignored": "value"}`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ta.handleHighlightBlockers(ctx, tt.arguments)

			if tt.wantError && err == nil {
				t.Errorf("handleHighlightBlockers() expected error for test %s but got none", tt.name)
			}
			if !tt.wantError && err != nil {
				t.Errorf("handleHighlightBlockers() unexpected error for test %s: %v", tt.name, err)
			}

			// Note: In actual implementation, you would mock the underlying
			// highlightBlockers method to return expected results
			_ = result
		})
	}
}

func TestHandleSuggestFocusAreas(t *testing.T) {
	ta := getTaskAgent()
	ctx := context.Background()

	tests := []struct {
		name      string
		arguments string
		wantError bool
	}{
		{
			name:      "Empty arguments",
			arguments: "",
			wantError: false,
		},
		{
			name:      "JSON arguments (ignored)",
			arguments: `{"ignored": "value"}`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ta.handleSuggestFocusAreas(ctx, tt.arguments)

			if tt.wantError && err == nil {
				t.Errorf("handleSuggestFocusAreas() expected error for test %s but got none", tt.name)
			}
			if !tt.wantError && err != nil {
				t.Errorf("handleSuggestFocusAreas() unexpected error for test %s: %v", tt.name, err)
			}

			// Note: In actual implementation, you would mock the underlying
			// suggestFocusAreas method to return expected results
			_ = result
		})
	}
}

// TestArgumentsStructures tests JSON marshaling/unmarshaling of argument structures
func TestArgumentsStructures(t *testing.T) {
	t.Run("FetchTasksArgs", func(t *testing.T) {
		complete := true
		priority := PRIORITY_HIGH
		effort := EFFORT_MEDIUM
		dueDate := "2025-10-03"
		project := "Test Project"

		args := FetchTasksArgs{
			Complete: &complete,
			Priority: &priority,
			Effort:   &effort,
			DueDate:  &dueDate,
			Project:  &project,
		}

		data, err := json.Marshal(args)
		if err != nil {
			t.Errorf("Failed to marshal FetchTasksArgs: %v", err)
		}

		var unmarshaled FetchTasksArgs
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Errorf("Failed to unmarshal FetchTasksArgs: %v", err)
		}

		if *unmarshaled.Complete != complete {
			t.Errorf("Complete mismatch: got %v, want %v", *unmarshaled.Complete, complete)
		}
		if *unmarshaled.Priority != priority {
			t.Errorf("Priority mismatch: got %v, want %v", *unmarshaled.Priority, priority)
		}
	})

	t.Run("GetTaskDetailsArgs", func(t *testing.T) {
		args := GetTaskDetailsArgs{
			TaskID: "test-task-id-123",
		}

		data, err := json.Marshal(args)
		if err != nil {
			t.Errorf("Failed to marshal GetTaskDetailsArgs: %v", err)
		}

		var unmarshaled GetTaskDetailsArgs
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Errorf("Failed to unmarshal GetTaskDetailsArgs: %v", err)
		}

		if unmarshaled.TaskID != args.TaskID {
			t.Errorf("TaskID mismatch: got %v, want %v", unmarshaled.TaskID, args.TaskID)
		}
	})

	t.Run("CreateTaskArgs", func(t *testing.T) {
		priority := PRIORITY_HIGH
		effort := EFFORT_HIGH
		dueDate := "2025-10-03"
		project := "Test Project"

		args := CreateTaskArgs{
			Title:    "Test Task",
			Priority: &priority,
			Effort:   &effort,
			DueDate:  &dueDate,
			Project:  &project,
		}

		data, err := json.Marshal(args)
		if err != nil {
			t.Errorf("Failed to marshal CreateTaskArgs: %v", err)
		}

		var unmarshaled CreateTaskArgs
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Errorf("Failed to unmarshal CreateTaskArgs: %v", err)
		}

		if unmarshaled.Title != args.Title {
			t.Errorf("Title mismatch: got %v, want %v", unmarshaled.Title, args.Title)
		}
		if *unmarshaled.Priority != priority {
			t.Errorf("Priority mismatch: got %v, want %v", *unmarshaled.Priority, priority)
		}
	})

	t.Run("UpdateTaskArgs", func(t *testing.T) {
		title := "Updated Task"
		priority := PRIORITY_MEDIUM
		effort := EFFORT_LOW
		dueDate := "2025-10-04"
		project := "Updated Project"

		args := UpdateTaskArgs{
			TaskID:   "test-task-id-123",
			Title:    &title,
			Priority: &priority,
			Effort:   &effort,
			DueDate:  &dueDate,
			Project:  &project,
		}

		data, err := json.Marshal(args)
		if err != nil {
			t.Errorf("Failed to marshal UpdateTaskArgs: %v", err)
		}

		var unmarshaled UpdateTaskArgs
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Errorf("Failed to unmarshal UpdateTaskArgs: %v", err)
		}

		if unmarshaled.TaskID != args.TaskID {
			t.Errorf("TaskID mismatch: got %v, want %v", unmarshaled.TaskID, args.TaskID)
		}
		if *unmarshaled.Title != title {
			t.Errorf("Title mismatch: got %v, want %v", *unmarshaled.Title, title)
		}
	})

	t.Run("CompleteTaskArgs", func(t *testing.T) {
		args := CompleteTaskArgs{
			TaskID: "test-task-id-123",
		}

		data, err := json.Marshal(args)
		if err != nil {
			t.Errorf("Failed to marshal CompleteTaskArgs: %v", err)
		}

		var unmarshaled CompleteTaskArgs
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Errorf("Failed to unmarshal CompleteTaskArgs: %v", err)
		}

		if unmarshaled.TaskID != args.TaskID {
			t.Errorf("TaskID mismatch: got %v, want %v", unmarshaled.TaskID, args.TaskID)
		}
	})
}
