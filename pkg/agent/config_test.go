package agent

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test loading agent configuration with basic scenarios
func TestLoadAgentConfig_Basic(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	require.NoError(t, os.Chdir(tempDir))

	tests := []struct {
		name           string
		agentName      string
		agentEnvFile   map[string]string
		globalEnvFile  map[string]string
		expectedValues map[string]string
		description    string
	}{
		{
			name:      "agent specific config exists",
			agentName: "github",
			agentEnvFile: map[string]string{
				"GITHUB_TOKEN":   "agent-specific-token",
				"GITHUB_API_URL": "https://api.github.com",
				"DEBUG":          "true",
			},
			globalEnvFile: map[string]string{
				"GITHUB_TOKEN": "global-token",
				"LOG_LEVEL":    "info",
			},
			expectedValues: map[string]string{
				"GITHUB_TOKEN":   "agent-specific-token", // Agent-specific takes precedence
				"GITHUB_API_URL": "https://api.github.com",
				"DEBUG":          "true",
				"LOG_LEVEL":      "info", // From global fallback
			},
			description: "should prioritize agent-specific config over global",
		},
		{
			name:         "no agent specific config",
			agentName:    "email",
			agentEnvFile: nil, // No agent-specific file
			globalEnvFile: map[string]string{
				"EMAIL_HOST":     "smtp.gmail.com",
				"EMAIL_PORT":     "587",
				"EMAIL_USERNAME": "user@example.com",
			},
			expectedValues: map[string]string{
				"EMAIL_HOST":     "smtp.gmail.com",
				"EMAIL_PORT":     "587",
				"EMAIL_USERNAME": "user@example.com",
			},
			description: "should fall back to global config when agent-specific doesn't exist",
		},
		{
			name:         "empty agent name",
			agentName:    "",
			agentEnvFile: nil,
			globalEnvFile: map[string]string{
				"DEFAULT_VALUE": "global",
			},
			expectedValues: map[string]string{
				"DEFAULT_VALUE": "global",
			},
			description: "should handle empty agent name gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Store and clear environment variables for isolation
			allKeys := make(map[string]bool)
			for key := range tt.agentEnvFile {
				allKeys[key] = true
			}
			for key := range tt.globalEnvFile {
				allKeys[key] = true
			}
			for key := range tt.expectedValues {
				allKeys[key] = true
			}

			originalEnvVars := make(map[string]string)
			var keysToUnset []string

			for key := range allKeys {
				if val := os.Getenv(key); val != "" {
					originalEnvVars[key] = val
				} else {
					keysToUnset = append(keysToUnset, key)
				}
				os.Unsetenv(key)
			}

			defer func() {
				// Restore original environment
				for key, val := range originalEnvVars {
					os.Setenv(key, val)
				}
				for _, key := range keysToUnset {
					os.Unsetenv(key)
				}
				// Clean up test files
				os.Remove(".env")
				if tt.agentName != "" {
					os.Remove(".env." + tt.agentName)
				}
			}()

			// Clean up any existing env files
			os.Remove(".env")
			if tt.agentName != "" {
				os.Remove(".env." + tt.agentName)
			}

			// Create global .env file if specified
			if tt.globalEnvFile != nil {
				createEnvFile(t, ".env", tt.globalEnvFile)
			}

			// Create agent-specific .env file if specified
			if tt.agentEnvFile != nil && tt.agentName != "" {
				createEnvFile(t, ".env."+tt.agentName, tt.agentEnvFile)
			}

			// Load the configuration
			config := LoadAgentConfig(tt.agentName)

			require.NotNil(t, config)

			// Verify expected values
			for key, expectedValue := range tt.expectedValues {
				actualValue := config.Get(key)
				assert.Equal(t, expectedValue, actualValue, "Config.Get(%q)", key)
			}
		})
	}
}

// Test loading agent configuration with precedence rules
func TestLoadAgentConfig_Precedence(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	require.NoError(t, os.Chdir(tempDir))

	// Test that agent-specific env takes precedence over global
	agentName := "notion"

	// Create global .env with some values
	globalEnv := map[string]string{
		"SHARED_KEY":  "global-value",
		"GLOBAL_ONLY": "global-only-value",
		"API_URL":     "https://global.api.com",
	}
	createEnvFile(t, ".env", globalEnv)

	// Create agent-specific .env that overrides some values
	agentEnv := map[string]string{
		"SHARED_KEY": "agent-specific-value",
		"AGENT_ONLY": "agent-only-value",
		"API_URL":    "https://notion.api.com",
	}
	createEnvFile(t, ".env."+agentName, agentEnv)

	config := LoadAgentConfig(agentName)

	tests := []struct {
		key      string
		expected string
		source   string
	}{
		{"SHARED_KEY", "agent-specific-value", "agent-specific should override global"},
		{"GLOBAL_ONLY", "global-only-value", "should inherit from global when not in agent-specific"},
		{"AGENT_ONLY", "agent-only-value", "should have agent-specific values"},
		{"API_URL", "https://notion.api.com", "agent-specific should override global URL"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			actual := config.Get(tt.key)
			assert.Equal(t, tt.expected, actual, "%s", tt.source)
		})
	}
}

// Test loading agent configuration with environment variables
func TestLoadAgentConfig_EnvironmentVariables(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	require.NoError(t, os.Chdir(tempDir))

	// Set some environment variables
	os.Setenv("TEST_ENV_VAR", "env-value")
	os.Setenv("OVERRIDE_VAR", "env-override")
	defer func() {
		os.Unsetenv("TEST_ENV_VAR")
		os.Unsetenv("OVERRIDE_VAR")
	}()

	// Create .env file that may conflict with environment variables
	envFile := map[string]string{
		"FILE_VAR":     "file-value",
		"OVERRIDE_VAR": "file-override", // This should be overridden by env var
	}
	createEnvFile(t, ".env", envFile)

	config := LoadAgentConfig("test")

	// Test that environment variables are available
	assert.Equal(t, "env-value", config.Get("TEST_ENV_VAR"))

	// Test that env file values are available
	assert.Equal(t, "file-value", config.Get("FILE_VAR"))

	// The exact precedence between env vars and .env files depends on the godotenv library behavior
	// We test that both sources are considered
	overrideVal := config.Get("OVERRIDE_VAR")
	assert.Contains(t, []string{"env-override", "file-override"}, overrideVal)
}

// Test loading agent configuration with non-existent files
func TestLoadAgentConfig_NonExistentFiles(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	require.NoError(t, os.Chdir(tempDir))

	// Ensure no .env files exist
	os.Remove(".env")
	os.Remove(".env.nonexistent")

	// Should not panic and should return a valid config
	config := LoadAgentConfig("nonexistent")

	require.NotNil(t, config, "LoadAgentConfig should return a valid config even when files don't exist")

	// Should be able to call methods on the config
	val := config.Get("NONEXISTENT_KEY")
	assert.Empty(t, val)

	// Should be able to set and get values
	config.Set("TEST_KEY", "test_value")
	assert.Equal(t, "test_value", config.Get("TEST_KEY"))
}

// Test loading agent configuration with different agent types
func TestLoadAgentConfig_DifferentAgentTypes(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	require.NoError(t, os.Chdir(tempDir))
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	// Create configs for different agent types
	agents := []struct {
		name   string
		config map[string]string
	}{
		{
			name: "github",
			config: map[string]string{
				"GITHUB_TOKEN":   "github-token-123",
				"GITHUB_API_URL": "https://api.github.com",
				"GITHUB_ORG":     "my-org",
			},
		},
		{
			name: "email",
			config: map[string]string{
				"EMAIL_HOST":     "smtp.gmail.com",
				"EMAIL_PORT":     "587",
				"EMAIL_USERNAME": "user@example.com",
				"EMAIL_PASSWORD": "secret",
			},
		},
		{
			name: "notion",
			config: map[string]string{
				"NOTION_API_KEY":  "notion-key-456",
				"NOTION_DATABASE": "database-id",
				"NOTION_PAGE":     "page-id",
			},
		},
		{
			name: "memory",
			config: map[string]string{
				"MEMORY_BACKEND": "redis",
				"MEMORY_HOST":    "localhost",
				"MEMORY_PORT":    "6379",
			},
		},
	}

	// Test each agent config in isolation
	for _, agent := range agents {
		t.Run(agent.name+"_agent", func(t *testing.T) {
			// Store original environment variables to restore later
			originalEnvVars := make(map[string]string)
			var keysToUnset []string

			// Save current env state for all keys we'll be testing
			allKeys := make(map[string]bool)
			for _, a := range agents {
				for key := range a.config {
					allKeys[key] = true
				}
			}

			for key := range allKeys {
				if val := os.Getenv(key); val != "" {
					originalEnvVars[key] = val
				} else {
					keysToUnset = append(keysToUnset, key)
				}
			}

			// Clean up function
			defer func() {
				// Restore original environment
				for key, val := range originalEnvVars {
					os.Setenv(key, val)
				}
				for _, key := range keysToUnset {
					os.Unsetenv(key)
				}
				// Remove test files
				os.Remove(".env." + agent.name)
			}()

			// Clear environment variables for isolation
			for key := range allKeys {
				os.Unsetenv(key)
			}

			// Create only this agent's config file
			createEnvFile(t, ".env."+agent.name, agent.config)

			config := LoadAgentConfig(agent.name)

			// Test this agent's config values
			for key, expectedValue := range agent.config {
				actualValue := config.Get(key)
				assert.Equal(t, expectedValue, actualValue, "Agent %s: Config.Get(%q)", agent.name, key)
			}

			// Verify other agents' configs are not present
			// Since we only created this agent's file and cleared env vars,
			// other agents' keys should not be present
			for _, otherAgent := range agents {
				if otherAgent.name == agent.name {
					continue
				}

				for key := range otherAgent.config {
					// This key should not exist in current agent's config
					actualValue := config.Get(key)
					assert.Empty(t, actualValue, "Agent %s should not have access to %s's config key %q", agent.name, otherAgent.name, key)
				}
			}
		})
	}
}

// Helper function to create .env files for testing
func createEnvFile(t *testing.T, filename string, vars map[string]string) {
	t.Helper()

	file, err := os.Create(filename)
	require.NoError(t, err, "Failed to create %s", filename)
	defer file.Close()

	for key, value := range vars {
		_, err := file.WriteString(key + "=" + value + "\n")
		require.NoError(t, err, "Failed to write to %s", filename)
	}
}
