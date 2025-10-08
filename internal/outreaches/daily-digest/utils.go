package outreach_dailydigest

import "time"

/* ---- HELPER FUNCTIONS ---- */

func intPointer(num int) *int {
	return &num
}

func boolPointer(val bool) *bool {
	return &val
}

func timePointer(t time.Time) *time.Time {
	return &t
}
