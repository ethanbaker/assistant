package outreach_notionschedule

import (
	notionapi "github.com/dstotijn/go-notion"
)

/* ---- CONSTANTS ---- */

const SCHEDULE_REMINDERS_ERROR_LIMIT = 10
const UPDATE_INTERVAL_MINUTES = 5

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

/* ---- HELPER FUNCTIONS ---- */

func boolPointer(val bool) *bool {
	return &val
}
