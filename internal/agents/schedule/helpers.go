package schedule

import (
	"strings"
	"time"

	gcal_service "github.com/ethanbaker/assistant/internal/services/gcal"
)

func normalizeArguments(arguments string) string {
	if strings.TrimSpace(arguments) == "" {
		return "{}"
	}

	return arguments
}

func isValidDate(dateStr string) bool {
	if strings.TrimSpace(dateStr) == "" {
		return false
	}

	_, err := time.Parse(gcal_service.DATE_FORMAT, dateStr)
	return err == nil
}
