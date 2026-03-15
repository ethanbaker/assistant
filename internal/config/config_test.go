// config_test.go - tests for the config package
package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoad tests the Load function
func TestLoad(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantPanic bool
	}{
		{
			name:      "valid env file",
			path:      "testing/.env.test",
			wantPanic: false,
		},
		{
			name:      "empty env file",
			path:      "testing/.env.empty",
			wantPanic: false,
		},
		{
			name:      "non-existent file",
			path:      "testing/.env.nonexistent",
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if (r != nil) != tt.wantPanic {
					t.Errorf("Load() panic = %v, wantPanic %v", r != nil, tt.wantPanic)
				}
			}()

			Load(tt.path)
		})
	}
}

// TestGetenv tests the Getenv function
func TestGetenv(t *testing.T) {
	// Load test environment
	Load("testing/.env.test")

	tests := []struct {
		name      string
		key       string
		wantValue string
		wantOk    bool
	}{
		{
			name:      "existing key",
			key:       "TEST_KEY",
			wantValue: "test_value",
			wantOk:    true,
		},
		{
			name:      "another existing key",
			key:       "DATABASE_URL",
			wantValue: "postgres://localhost:5432/testdb",
			wantOk:    true,
		},
		{
			name:      "non-existent key",
			key:       "NONEXISTENT_KEY",
			wantValue: "",
			wantOk:    false,
		},
		{
			name:      "empty value key",
			key:       "EMPTY_VALUE",
			wantValue: "",
			wantOk:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValue, gotOk := Getenv(tt.key)
			if gotValue != tt.wantValue {
				t.Errorf("Getenv() gotValue = %v, want %v", gotValue, tt.wantValue)
			}
			if gotOk != tt.wantOk {
				t.Errorf("Getenv() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

// TestMustGetenv tests the MustGetenv function
func TestMustGetenv(t *testing.T) {
	// Load test environment
	Load("testing/.env.test")

	tests := []struct {
		name      string
		key       string
		want      string
		wantPanic bool
	}{
		{
			name:      "existing key",
			key:       "TEST_KEY",
			want:      "test_value",
			wantPanic: false,
		},
		{
			name:      "api key",
			key:       "API_KEY",
			want:      "secret123",
			wantPanic: false,
		},
		{
			name:      "non-existent key",
			key:       "MUST_FAIL_KEY",
			want:      "",
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if (r != nil) != tt.wantPanic {
					t.Errorf("MustGetenv() panic = %v, wantPanic %v", r != nil, tt.wantPanic)
				}
			}()

			got := MustGetenv(tt.key)
			if !tt.wantPanic && got != tt.want {
				t.Errorf("MustGetenv() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetenvValue tests the GetenvValue function
func TestGetenvValue(t *testing.T) {
	// Load test environment
	Load("testing/.env.test")

	tests := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "existing key",
			key:  "TEST_KEY",
			want: "test_value",
		},
		{
			name: "port key",
			key:  "PORT",
			want: "8080",
		},
		{
			name: "non-existent key returns empty string",
			key:  "NONEXISTENT",
			want: "",
		},
		{
			name: "empty value returns empty string",
			key:  "EMPTY_VALUE",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetenvValue(tt.key)
			if got != tt.want {
				t.Errorf("GetenvValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetenvWithDefault tests the GetenvWithDefault function
func TestGetenvWithDefault(t *testing.T) {
	// Load test environment
	Load("testing/.env.test")

	tests := []struct {
		name         string
		key          string
		defaultValue string
		want         string
	}{
		{
			name:         "existing key ignores default",
			key:          "TEST_KEY",
			defaultValue: "default_value",
			want:         "test_value",
		},
		{
			name:         "non-existent key uses default",
			key:          "MISSING_KEY",
			defaultValue: "default_value",
			want:         "default_value",
		},
		{
			name:         "empty default value",
			key:          "ANOTHER_MISSING_KEY",
			defaultValue: "",
			want:         "",
		},
		{
			name:         "existing key with empty value",
			key:          "EMPTY_VALUE",
			defaultValue: "default_value",
			want:         "",
		},
		{
			name:         "debug flag",
			key:          "DEBUG",
			defaultValue: "false",
			want:         "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetenvWithDefault(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("GetenvWithDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestKeyExists tests the KeyExists function
func TestKeyExists(t *testing.T) {
	// Load test environment
	Load("testing/.env.test")

	tests := []struct {
		name string
		key  string
		want bool
	}{
		{
			name: "existing key",
			key:  "TEST_KEY",
			want: true,
		},
		{
			name: "another existing key",
			key:  "DATABASE_URL",
			want: true,
		},
		{
			name: "non-existent key",
			key:  "NONEXISTENT_KEY",
			want: false,
		},
		{
			name: "empty value key exists",
			key:  "EMPTY_VALUE",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := KeyExists(tt.key)
			if got != tt.want {
				t.Errorf("KeyExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestLoadMultipleTimes tests loading different env files
func TestLoadMultipleTimes(t *testing.T) {
	// Load first env file
	Load("testing/.env.test")
	
	// Verify first file is loaded
	if val := GetenvValue("TEST_KEY"); val != "test_value" {
		t.Errorf("First load failed, got %v", val)
	}

	// Load second env file with different values
	Load("testing/.env.duplicate")

	// Verify second file is loaded and first values are replaced
	if val := GetenvValue("NORMAL_KEY"); val != "normal_value" {
		t.Errorf("Second load failed, got %v", val)
	}

	// Old keys should not exist anymore
	if KeyExists("TEST_KEY") {
		t.Error("Old keys should not exist after loading new env file")
	}
}

// TestLoadWithAbsolutePath tests loading with absolute path
func TestLoadWithAbsolutePath(t *testing.T) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Construct absolute path
	absPath := filepath.Join(cwd, "testing", ".env.test")

	// Load using absolute path
	Load(absPath)

	// Verify it loaded correctly
	if val := GetenvValue("TEST_KEY"); val != "test_value" {
		t.Errorf("Load with absolute path failed, got %v", val)
	}
}

// TestEmptyEnvFile tests loading an empty env file
func TestEmptyEnvFile(t *testing.T) {
	// Load empty env file
	Load("testing/.env.empty")

	// Any key should not exist
	if KeyExists("ANY_KEY") {
		t.Error("Empty env file should not contain any keys")
	}

	// GetenvValue should return empty string
	if val := GetenvValue("ANY_KEY"); val != "" {
		t.Errorf("Expected empty string, got %v", val)
	}
}

// TestSpecialCharacters tests handling of special characters
func TestSpecialCharacters(t *testing.T) {
	// Load env file with special characters
	Load("testing/.env.duplicate")

	tests := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "special characters",
			key:  "SPECIAL_CHARS",
			want: "!@#$%^&*()",
		},
		{
			name: "quoted value",
			key:  "QUOTED_VALUE",
			want: "quoted text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetenvValue(tt.key)
			if got != tt.want {
				t.Errorf("GetenvValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

// BenchmarkGetenv benchmarks the Getenv function
func BenchmarkGetenv(b *testing.B) {
	Load("testing/.env.test")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Getenv("TEST_KEY")
	}
}

// BenchmarkGetenvValue benchmarks the GetenvValue function
func BenchmarkGetenvValue(b *testing.B) {
	Load("testing/.env.test")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetenvValue("TEST_KEY")
	}
}

// BenchmarkGetenvWithDefault benchmarks the GetenvWithDefault function
func BenchmarkGetenvWithDefault(b *testing.B) {
	Load("testing/.env.test")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetenvWithDefault("TEST_KEY", "default")
	}
}

// BenchmarkKeyExists benchmarks the KeyExists function
func BenchmarkKeyExists(b *testing.B) {
	Load("testing/.env.test")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		KeyExists("TEST_KEY")
	}
}
