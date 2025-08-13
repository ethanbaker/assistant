package utils

import (
	"os"
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	t.Run("with nil values", func(t *testing.T) {
		config := NewConfig(nil)
		require.NotNil(t, config)
		assert.Len(t, config.Keys(), 0)
	})

	t.Run("with empty map", func(t *testing.T) {
		config := NewConfig(map[string]string{})
		assert.Len(t, config.Keys(), 0)
	})

	t.Run("with values", func(t *testing.T) {
		values := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}
		config := NewConfig(values)

		assert.Equal(t, "value1", config.Get("key1"))
		assert.Equal(t, "value2", config.Get("key2"))

		// Verify it's a copy, not a reference
		values["key1"] = "modified"
		assert.NotEqual(t, "modified", config.Get("key1"))
	})
}

func TestNewConfigFromEnv(t *testing.T) {
	// Create a temporary .env file for testing
	envContent := "TEST_KEY1=test_value1\nTEST_KEY2=test_value2\n"
	tmpFile, err := os.CreateTemp("", "test_env_*.env")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(envContent)
	require.NoError(t, err)
	tmpFile.Close()

	config := NewConfigFromEnv(tmpFile.Name())

	require.NotNil(t, config)
}

func TestConfigGet(t *testing.T) {
	config := NewConfig(map[string]string{
		"existing": "value",
		"empty":    "",
	})

	t.Run("existing key", func(t *testing.T) {
		assert.Equal(t, "value", config.Get("existing"))
	})

	t.Run("non-existing key", func(t *testing.T) {
		assert.Empty(t, config.Get("missing"))
	})

	t.Run("empty value key", func(t *testing.T) {
		assert.Empty(t, config.Get("empty"))
	})
}

func TestConfigGetWithDefault(t *testing.T) {
	config := NewConfig(map[string]string{
		"existing": "value",
		"empty":    "",
	})

	t.Run("existing key", func(t *testing.T) {
		got := config.GetWithDefault("existing", "default")
		assert.Equal(t, "value", got)
	})

	t.Run("non-existing key", func(t *testing.T) {
		got := config.GetWithDefault("missing", "default")
		assert.Equal(t, "default", got)
	})

	t.Run("empty value key", func(t *testing.T) {
		got := config.GetWithDefault("empty", "default")
		assert.Equal(t, "default", got)
	})
}

func TestConfigGetBool(t *testing.T) {
	config := NewConfig(map[string]string{
		"true_bool":      "true",
		"false_bool":     "false",
		"true_1":         "1",
		"false_0":        "0",
		"true_yes":       "yes",
		"false_no":       "no",
		"true_on":        "on",
		"false_off":      "off",
		"true_enabled":   "enabled",
		"false_disabled": "disabled",
		"invalid":        "invalid_bool",
		"empty":          "",
	})

	tests := []struct {
		key      string
		expected bool
	}{
		{"true_bool", true},
		{"false_bool", false},
		{"true_1", true},
		{"false_0", false},
		{"true_yes", true},
		{"false_no", false},
		{"true_on", true},
		{"false_off", false},
		{"true_enabled", true},
		{"false_disabled", false},
		{"invalid", false},
		{"empty", false},
		{"missing", false},
	}

	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {
			got := config.GetBool(test.key)
			assert.Equal(t, test.expected, got, "GetBool(%s)", test.key)
		})
	}
}

func TestConfigGetBoolWithDefault(t *testing.T) {
	config := NewConfig(map[string]string{
		"true_bool": "true",
		"empty":     "",
	})

	t.Run("existing key", func(t *testing.T) {
		got := config.GetBoolWithDefault("true_bool", false)
		assert.True(t, got)
	})

	t.Run("non-existing key with default true", func(t *testing.T) {
		got := config.GetBoolWithDefault("missing", true)
		assert.True(t, got)
	})

	t.Run("non-existing key with default false", func(t *testing.T) {
		got := config.GetBoolWithDefault("missing", false)
		assert.False(t, got)
	})

	t.Run("empty value key", func(t *testing.T) {
		got := config.GetBoolWithDefault("empty", true)
		assert.False(t, got) // Expected false (parsed)
	})
}

func TestConfigGetInt(t *testing.T) {
	config := NewConfig(map[string]string{
		"valid_int":   "42",
		"zero":        "0",
		"negative":    "-10",
		"invalid_int": "not_a_number",
		"empty":       "",
	})

	tests := []struct {
		key      string
		expected int
	}{
		{"valid_int", 42},
		{"zero", 0},
		{"negative", -10},
		{"invalid_int", 0},
		{"empty", 0},
		{"missing", 0},
	}

	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {
			got := config.GetInt(test.key)
			assert.Equal(t, test.expected, got, "GetInt(%s)", test.key)
		})
	}
}

func TestConfigGetIntWithDefault(t *testing.T) {
	config := NewConfig(map[string]string{
		"valid_int": "42",
		"empty":     "",
	})

	t.Run("existing key", func(t *testing.T) {
		got := config.GetIntWithDefault("valid_int", 999)
		assert.Equal(t, 42, got)
	})

	t.Run("non-existing key", func(t *testing.T) {
		got := config.GetIntWithDefault("missing", 999)
		assert.Equal(t, 999, got)
	})

	t.Run("empty value key", func(t *testing.T) {
		got := config.GetIntWithDefault("empty", 999)
		assert.Equal(t, 0, got) // Expected 0 (parsed)
	})
}

func TestConfigSet(t *testing.T) {
	config := NewConfig(map[string]string{})

	config.Set("new_key", "new_value")
	assert.Equal(t, "new_value", config.Get("new_key"))

	// Test overwriting
	config.Set("new_key", "updated_value")
	assert.Equal(t, "updated_value", config.Get("new_key"))
}

func TestConfigSetBool(t *testing.T) {
	config := NewConfig(map[string]string{})

	config.SetBool("bool_true", true)
	config.SetBool("bool_false", false)

	assert.True(t, config.GetBool("bool_true"))
	assert.False(t, config.GetBool("bool_false"))

	// Verify string representation
	assert.Equal(t, "true", config.Get("bool_true"))
	assert.Equal(t, "false", config.Get("bool_false"))
}

func TestConfigSetInt(t *testing.T) {
	config := NewConfig(map[string]string{})

	config.SetInt("int_positive", 42)
	config.SetInt("int_zero", 0)
	config.SetInt("int_negative", -10)

	assert.Equal(t, 42, config.GetInt("int_positive"))
	assert.Equal(t, 0, config.GetInt("int_zero"))
	assert.Equal(t, -10, config.GetInt("int_negative"))

	// Verify string representation
	assert.Equal(t, "42", config.Get("int_positive"))
}

func TestConfigDelete(t *testing.T) {
	config := NewConfig(map[string]string{
		"key1": "value1",
		"key2": "value2",
	})

	assert.True(t, config.Has("key1"))

	config.Delete("key1")

	assert.False(t, config.Has("key1"))
	assert.Empty(t, config.Get("key1"))

	// Ensure other keys are unaffected
	assert.True(t, config.Has("key2"))

	// Deleting non-existent key should not panic
	config.Delete("non_existent")
}

func TestConfigHas(t *testing.T) {
	config := NewConfig(map[string]string{
		"existing": "value",
		"empty":    "",
	})

	assert.True(t, config.Has("existing"))
	assert.True(t, config.Has("empty"))
	assert.False(t, config.Has("missing"))
}

func TestConfigKeys(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		config := NewConfig(map[string]string{})
		keys := config.Keys()
		assert.Len(t, keys, 0)
	})

	t.Run("config with keys", func(t *testing.T) {
		config := NewConfig(map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		})
		keys := config.Keys()

		assert.Len(t, keys, 3)

		// Sort for consistent comparison
		sort.Strings(keys)
		expected := []string{"key1", "key2", "key3"}
		sort.Strings(expected)

		assert.Equal(t, expected, keys)
	})
}

func TestConfigToMap(t *testing.T) {
	original := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	config := NewConfig(original)

	result := config.ToMap()

	assert.Equal(t, original, result)

	// Verify it's a copy, not a reference
	result["key1"] = "modified"
	assert.NotEqual(t, "modified", config.Get("key1"))
}

func TestConfigMerge(t *testing.T) {
	t.Run("merge with nil", func(t *testing.T) {
		config := NewConfig(map[string]string{"key1": "value1"})
		config.Merge(nil)
		// Should not panic and should be unchanged
		assert.Equal(t, "value1", config.Get("key1"))
	})

	t.Run("merge with another config", func(t *testing.T) {
		config1 := NewConfig(map[string]string{
			"key1": "value1",
			"key2": "value2",
		})
		config2 := NewConfig(map[string]string{
			"key2": "new_value2", // Override
			"key3": "value3",     // New key
		})

		config1.Merge(config2)

		assert.Equal(t, "value1", config1.Get("key1"))
		assert.Equal(t, "new_value2", config1.Get("key2"))
		assert.Equal(t, "value3", config1.Get("key3"))
	})
}

func TestConfigSyncWithEnv(t *testing.T) {
	// Set up environment variables
	os.Setenv("TEST_SYNC_KEY1", "env_value1")
	os.Setenv("TEST_SYNC_KEY2", "env_value2")
	defer func() {
		os.Unsetenv("TEST_SYNC_KEY1")
		os.Unsetenv("TEST_SYNC_KEY2")
	}()

	config := NewConfig(map[string]string{
		"TEST_SYNC_KEY1": "config_value1",
		"OTHER_KEY":      "other_value",
	})

	config.SyncWithEnv()

	// Key that exists in both config and env should be updated
	assert.Equal(t, "env_value1", config.Get("TEST_SYNC_KEY1"))

	// Key that doesn't exist in env should remain unchanged
	assert.Equal(t, "other_value", config.Get("OTHER_KEY"))

	// Key that exists in env but not in config should not be added
	assert.False(t, config.Has("TEST_SYNC_KEY2"))
}

func TestConfigClone(t *testing.T) {
	original := NewConfig(map[string]string{
		"key1": "value1",
		"key2": "value2",
	})

	clone := original.Clone()

	// Verify clone has same values
	assert.Equal(t, "value1", clone.Get("key1"))
	assert.Equal(t, "value2", clone.Get("key2"))

	// Verify independence
	clone.Set("key1", "modified")
	assert.NotEqual(t, "modified", original.Get("key1"))

	original.Set("key2", "modified_original")
	assert.NotEqual(t, "modified_original", clone.Get("key2"))

	// Verify new keys are independent
	clone.Set("clone_only", "clone_value")
	assert.False(t, original.Has("clone_only"))
}

func TestConfigThreadSafety(t *testing.T) {
	config := NewConfig(map[string]string{
		"counter": "0",
	})

	const numGoroutines = 100
	const operationsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Start multiple goroutines that read and write concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				// Mix of read and write operations
				config.Set("test_key_"+string(rune(id)), "value")
				config.Get("test_key_" + string(rune(id)))
				config.Has("test_key_" + string(rune(id)))
				config.GetBool("counter")
				config.GetInt("counter")
				config.Keys()
				config.ToMap()
			}
		}(i)
	}

	wg.Wait()
	// Test passes if no data races occur
}

func TestConfigConcurrentMerge(t *testing.T) {
	config1 := NewConfig(map[string]string{"base": "value"})
	config2 := NewConfig(map[string]string{"merge": "value"})

	var wg sync.WaitGroup
	wg.Add(2)

	// Concurrent merge and clone operations
	go func() {
		defer wg.Done()
		for range 50 {
			config1.Merge(config2)
		}
	}()

	go func() {
		defer wg.Done()
		for range 50 {
			_ = config1.Clone()
		}
	}()

	wg.Wait()
	// Test passes if no data races occur
}
