package gcal

import (
	"time"
)

// IsValidDate checks if a date string is in the correct format
func IsValidDate(dateStr string) bool {
	// Allow empty dates
	if dateStr == "" {
		return true
	}

	_, err := time.Parse(DATE_FORMAT, dateStr)
	return err == nil
}
