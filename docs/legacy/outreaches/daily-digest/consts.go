package outreach_dailydigest

import (
	"time"

	notionapi "github.com/dstotijn/go-notion"
)

/* ---- CONSTANTS ---- */

// Populated on init
var NEWS_PROMPT string

const RETRY_LIMIT = 10

var NORMAL_TASKS = NotionDatabase{
	ID: "",
	Query: notionapi.DatabaseQuery{
		Filter: &notionapi.DatabaseQueryFilter{
			And: []notionapi.DatabaseQueryFilter{
				// 'Complete' is unchecked
				{
					Property: "Complete",
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
							Equals: boolPointer(false),
						},
					},
				},
				// And 'Canceled' is unchecked
				{
					Property: "Canceled",
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Formula: &notionapi.FormulaDatabaseQueryFilter{
							Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
								Equals: boolPointer(false),
							},
						},
					},
				},
				// And 'Priority' is not critical
				{
					Property: "Priority",
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Select: &notionapi.SelectDatabaseQueryFilter{
							DoesNotEqual: "Critical",
						},
					},
				},
				{
					Or: []notionapi.DatabaseQueryFilter{
						// And 'Rank' is greater than five and date is on or before 1 week from now
						{
							Property: "Rank",
							DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
								Formula: &notionapi.FormulaDatabaseQueryFilter{
									Number: &notionapi.NumberDatabaseQueryFilter{
										GreaterThanOrEqualTo: intPointer(5),
									},
								},
							},
						},
						// Or 'Date' is on or before one week from now
						{
							Property: "Date",
							DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
								Date: &notionapi.DatePropertyFilter{
									NextWeek: &struct{}{},
								},
							},
						},
						{
							Property: "Date",
							DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
								Date: &notionapi.DatePropertyFilter{
									OnOrBefore: timePointer(time.Now()),
								},
							},
						},
					},
				},
			},
		},
		// Sort by ascending date
		Sorts: []notionapi.DatabaseQuerySort{
			{
				Property:  "Date",
				Direction: notionapi.SortDirAsc,
			},
		},
	},
}

var CRITICAL_TASKS = NotionDatabase{
	ID: "",
	Query: notionapi.DatabaseQuery{
		Filter: &notionapi.DatabaseQueryFilter{
			And: []notionapi.DatabaseQueryFilter{
				// 'Priority' equals 'Critical'
				{
					Property: "Priority",
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Select: &notionapi.SelectDatabaseQueryFilter{
							Equals: "Critical",
						},
					},
				},
				// And 'Canceled' is unchecked
				{
					Property: "Canceled",
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Formula: &notionapi.FormulaDatabaseQueryFilter{
							Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
								Equals: boolPointer(false),
							},
						},
					},
				},
				// And 'Complete' is unchecked
				{
					Property: "Complete",
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
							Equals: boolPointer(false),
						},
					},
				},
			},
		},
		// Sort by descending effort
		Sorts: []notionapi.DatabaseQuerySort{
			{
				Property:  "Effort",
				Direction: notionapi.SortDirDesc,
			},
		},
	},
}

var SCHEDULE_ITEMS = NotionDatabase{
	ID: "",
	Query: notionapi.DatabaseQuery{
		Filter: &notionapi.DatabaseQueryFilter{
			// 'Day' is checked
			Property: "Day",
			DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
				Formula: &notionapi.FormulaDatabaseQueryFilter{
					Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
						Equals: boolPointer(true),
					},
				},
			},
		},
		// Sort by ascending date
		Sorts: []notionapi.DatabaseQuerySort{
			{
				Property:  "Date",
				Direction: notionapi.SortDirAsc,
			},
		},
	},
}

var RECURRING_TASKS = NotionDatabase{
	ID: "",
	Query: notionapi.DatabaseQuery{
		Filter: &notionapi.DatabaseQueryFilter{
			And: []notionapi.DatabaseQueryFilter{
				// 'Active' is checked
				{
					Property: "Active",
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
							Equals: boolPointer(true),
						},
					},
				},
				// And 'Upcoming' is checked
				{
					Property: "Upcoming",
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Formula: &notionapi.FormulaDatabaseQueryFilter{
							Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
								Equals: boolPointer(true),
							},
						},
					},
				},
				// And 'Done' is unchecked
				{
					Property: "Done",
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
							Equals: boolPointer(false),
						},
					},
				},
				// And 'Type' is ___
				{
					Property: "Type",
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Select: &notionapi.SelectDatabaseQueryFilter{
							Equals: "",
						},
					},
				},
			},
		},
		// Sort by descending type and ascending name
		Sorts: []notionapi.DatabaseQuerySort{
			{
				Property:  "Name",
				Direction: notionapi.SortDirAsc,
			},
		},
	},
}
