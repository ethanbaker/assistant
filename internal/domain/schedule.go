package domain

// ScheduleType identifies the concrete schedule payload format.
type ScheduleType string

const (
	ScheduleCron   ScheduleType = "cron"
	ScheduleCustom ScheduleType = "custom"
)

// ScheduleEnvelope is used for discriminator-based unmarshaling.
type ScheduleEnvelope struct {
	Type ScheduleType `json:"type"`
}

// CronSchedule stores cron-based schedule settings.
type CronSchedule struct {
	Type       ScheduleType `json:"type"`
	CronString string       `json:"cron_string"`
}

// CustomSchedule stores simple interval schedule settings.
type CustomSchedule struct {
	Type       ScheduleType `json:"type"`
	IntervalMs int64        `json:"interval_ms"`
	OffsetMs   int64        `json:"offset_ms"`
}
