package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadPrompt(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	// Create prompts directory
	err := os.Mkdir("prompts", 0755)
	require.NoError(t, err)

	// Test case 1: Load from exact path prompts/test-agent.txt
	testContent1 := "You are a helpful assistant.\nProvide clear and concise answers."
	testFile1 := filepath.Join("prompts", "test-agent.txt")
	err = os.WriteFile(testFile1, []byte(testContent1), 0644)
	require.NoError(t, err)

	content, err := LoadPrompt(testFile1)
	require.NoError(t, err)
	assert.Equal(t, testContent1, content)

	// Test case 2: Load from exact path prompts/markdown-agent.md
	testContent2 := "# Assistant Instructions\n\nYou are a specialized agent."
	testFile2 := filepath.Join("prompts", "markdown-agent.md")
	err = os.WriteFile(testFile2, []byte(testContent2), 0644)
	require.NoError(t, err)

	content, err = LoadPrompt(testFile2)
	require.NoError(t, err)
	assert.Equal(t, testContent2, content)

	// Test case 3: File not found
	_, err = LoadPrompt("nonexistent-file.txt")
	assert.Error(t, err)

	// Test case 4: Load from root directory with exact path
	testContent3 := "Root level prompt content"
	testFile3 := "root-agent.txt"
	err = os.WriteFile(testFile3, []byte(testContent3), 0644)
	require.NoError(t, err)

	content, err = LoadPrompt(testFile3)
	require.NoError(t, err)
	assert.Equal(t, testContent3, content)
}

func TestLoadPromptWithFallback(t *testing.T) {
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	fallbackContent := "This is a fallback prompt"

	// Test case 1: File exists
	os.Mkdir("prompts", 0755)
	testContent := "Actual prompt content"
	testFile := filepath.Join("prompts", "existing-agent.txt")
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	content := LoadPromptWithFallback(testFile, fallbackContent)
	assert.Equal(t, testContent, content)

	// Test case 2: File doesn't exist, use fallback
	content = LoadPromptWithFallback("nonexistent-file.txt", fallbackContent)
	assert.Equal(t, fallbackContent, content)
}
