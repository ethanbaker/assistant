package agent

import "github.com/ethanbaker/assistant/pkg/utils"

// LoadAgentConfig loads configuration for a specific agent
// It tries to load agent-specific .env file first, then falls back to global .env
func LoadAgentConfig(agentName string) *utils.Config {
	agentEnvFile := ".env." + agentName
	return utils.NewConfigFromEnv(agentEnvFile, ".env")
}
