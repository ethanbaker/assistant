package outreach_dailydigest_test

import (
	"testing"

	outreach_dailydigest "github.com/ethanbaker/assistant/internal/outreaches/daily-digest"
	"github.com/ethanbaker/assistant/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestCreateDailyDigest(t *testing.T) {
	assert := assert.New(t)

	// Create test config
	envFile := ".env.test"
	cfg := utils.NewConfigFromEnv(envFile)

	// Initialize outreach
	err := outreach_dailydigest.Init(cfg)
	assert.Nil(err)

	// Run outreach
	output := outreach_dailydigest.CreateDailyDigest(cfg)
	assert.NotNil(output)
	t.Logf("Result: %v", output)
}
