package notion

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
	RECURRING_COLUMN_ACTIVE     = "Active"
	RECURRING_COLUMN_UPCOMING   = "Upcoming - Offset"
	RECURRING_COLUMN_DONE       = "Done"
	RECURRING_COLUMN_TITLE      = "Name"
	RECURRING_COLUMN_LAST_DONE  = "Last Done"
	RECURRING_COLUMN_DATE       = "Due Date"
	RECURRING_COLUMN_TYPE       = "Type"
	RECURRING_COLUMN_CONNECTION = "Connection Label"

	// Schedule Item Database Column Names
	SCHEDULE_COLUMN_TITLE    = "Name"
	SCHEDULE_COLUMN_DATE     = "Date"
	SCHEDULE_COLUMN_TIMESPAN = "Timespan"
	SCHEDULE_COLUMN_PROJECT  = "Project Label"

	// Date formats
	DATE_FORMAT        = "2006-01-02"
	PRETTY_DATE_FORMAT = "Jan 2, 2006"
	TIME_FORMAT        = "15:04"
	PRETTY_TIME_FORMAT = "3:04 PM"
)
