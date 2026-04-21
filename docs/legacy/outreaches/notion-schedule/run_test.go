package outreach_notionschedule

import (
	"testing"

	"github.com/ethanbaker/assistant/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestNotionScheduleReminder(t *testing.T) {
	assert := assert.New(t)

	// Create test config
	envFile := ".env.test"
	cfg := utils.NewConfigFromEnv(envFile)

	// Initialize outreach
	err := Init(cfg)
	assert.Nil(err)

	// Fetch events
	fetchNotionEvents(cfg)

	// Test when no events are active
	output := NotionScheduleReminder(cfg)
	t.Logf("Result: %v", output)
}
