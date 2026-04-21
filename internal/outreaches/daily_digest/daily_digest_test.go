package dailydigest

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/ethanbaker/assistant/internal/services/gcal"
	"github.com/ethanbaker/assistant/internal/services/notion"
	"github.com/ethanbaker/assistant/pkg/config"
	"github.com/goccy/go-yaml"
)

func TestRunDailyDigest(t *testing.T) {
	config.Load(".env.test")

	calendarConfigPath := config.MustGetenv("GOOGLE_CALENDAR_CONFIG_FILE")
	yamlFile, err := os.ReadFile(calendarConfigPath)
	if err != nil {
		t.Fatalf("failed to read calendar config: %v", err)
	}

	var calendarCfg gcal.CalendarServiceConfig
	if err := yaml.Unmarshal(yamlFile, &calendarCfg); err != nil {
		t.Fatalf("failed to parse calendar config: %v", err)
	}

	calendarService, err := gcal.NewCalendarService(calendarCfg)
	if err != nil {
		t.Fatalf("failed to create calendar service: %v", err)
	}

	notionService, err := notion.NewNotionTaskService(notion.NotionTaskServiceConfig{
		APIToken:            config.MustGetenv("NOTION_API_TOKEN"),
		TasksDatabaseID:     config.MustGetenv("NOTION_DATABASE_TASKS_ID"),
		RecurringDatabaseID: config.MustGetenv("NOTION_DATABASE_RECURRING_ID"),
		ScheduleDatabaseID:  config.MustGetenv("NOTION_DATABASE_SCHEDULE_ID"),
		Timezone:            config.MustGetenv("TIMEZONE"),
	})
	if err != nil {
		t.Fatalf("failed to create notion service: %v", err)
	}

	digest := NewDailyDigest(calendarService, notionService)
	output, err := digest.RunDailyDigest(context.Background(), nil)
	if err != nil {
		t.Fatalf("RunDailyDigest failed: %v", err)
	}

	fmt.Println(output)
}
