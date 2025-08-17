package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// findProjectRoot searches for the project root directory by looking for go.mod
func findProjectRoot() (string, error) {
	// Start from current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up the directory tree looking for go.mod
	for {
		goModPath := filepath.Join(currentDir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return currentDir, nil
		}

		// Move up one directory
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// Reached root directory
			break
		}
		currentDir = parentDir
	}

	return "", fmt.Errorf("could not find project root (go.mod not found)")
}

// LoadPrompt loads prompt instructions from a specific file path
// If the path is relative, it will be resolved from the project root
func LoadPrompt(filePath string) (string, error) {
	var fullPath string

	// If path is relative, resolve it from project root
	if !filepath.IsAbs(filePath) {
		projectRoot, err := findProjectRoot()
		if err != nil {
			return "", fmt.Errorf("failed to find project root: %w", err)
		}
		fullPath = filepath.Join(projectRoot, filePath)
	} else {
		fullPath = filePath
	}

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", fullPath)
	}

	// Read file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", fullPath, err)
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
