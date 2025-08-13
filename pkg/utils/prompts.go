package utils

import (
	"fmt"
	"os"
	"strings"
)

// LoadPrompt loads prompt instructions from a specific file path
// The path must be exact - no fallback searching is performed
func LoadPrompt(filePath string) (string, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", filePath)
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Trim whitespace and return
	return strings.TrimSpace(string(content)), nil
}

// LoadPromptWithFallback loads prompt instructions from a specific file path with a fallback
// If the file is not found, it returns the fallback string
func LoadPromptWithFallback(filePath, fallback string) string {
	if content, err := LoadPrompt(filePath); err == nil {
		return content
	}
	return fallback
}
