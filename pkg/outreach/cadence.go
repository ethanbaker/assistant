package outreach

/** Cadence handles scheduling repeat tasks */

// CadenceType represents the type of scheduling cadence for a task
type CadenceType string

const (
	// CronCadence schedules tasks using cron expressions
	CronCadence CadenceType = "cron"

	// SunriseCadence schedules tasks to run at local sunrise
	SunriseCadence CadenceType = "sunrise"

	// SunsetCadence schedules tasks to run at local sunset
	SunsetCadence CadenceType = "sunset"
)

// ValidateCadenceType checks if the given cadence type is valid
func ValidateCadenceType(cadence CadenceType) bool {
	switch cadence {
	case CronCadence, SunriseCadence, SunsetCadence:
		return true
	default:
		return false
	}
}
