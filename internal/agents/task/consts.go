package task

const (
	// Priority Select Values
	PRIORITY_NONE     = "None"
	PRIORITY_LOW      = "Low (1)"
	PRIORITY_MEDIUM   = "Medium (2)"
	PRIORITY_HIGH     = "High (3)"
	PRIORITY_CRITICAL = "Critical"

	// Effort Select Values
	EFFORT_LOW    = "Low (1)"
	EFFORT_MEDIUM = "Medium (2)"
	EFFORT_HIGH   = "High (3)"

	// Task Database Column Names
	COLUMN_COMPLETE = "Complete"
	COLUMN_PRIORITY = "Priority"
	COLUMN_EFFORT   = "Effort"
	COLUMN_DATE     = "Date"
	COLUMN_PROJECT  = "Project Label"
	COLUMN_TITLE    = "Name"
	COLUMN_CANCELED = "Canceled"
	COLUMN_RANK     = "Rank"

	// Recurring Task Database Column Names
	RECURRING_COLUMN_ACTIVE   = "Active"
	RECURRING_COLUMN_UPCOMING = "Upcoming"
	RECURRING_COLUMN_DONE     = "Done"
	RECURRING_COLUMN_TITLE    = "Name"
)
