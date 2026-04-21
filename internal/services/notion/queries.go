package notion

import (
	"time"

	notionapi "github.com/dstotijn/go-notion"
)

func (ns *NotionTaskService) buildFetchTasksQuery(args FetchTasksArgs) *notionapi.DatabaseQuery {
	filters := []notionapi.DatabaseQueryFilter{}

	// Always filter canceled tasks
	filters = append(filters, notionapi.DatabaseQueryFilter{
		Property: COLUMN_CANCELED,
		DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
			Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
				Equals: new(false),
			},
		},
	})

	// Filter by complete status
	if args.Complete != nil {
		filters = append(filters, notionapi.DatabaseQueryFilter{
			Property: COLUMN_COMPLETE,
			DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
				Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
					Equals: args.Complete,
				},
			},
		})
	}

	// Filter by priority
	if args.Priority != nil && *args.Priority != "" {
		filters = append(filters, notionapi.DatabaseQueryFilter{
			Property: COLUMN_PRIORITY,
			DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
				Select: &notionapi.SelectDatabaseQueryFilter{
					Equals: *args.Priority,
				},
			},
		})
	}

	// Filter by effort
	if args.Effort != nil && *args.Effort != "" {
		filters = append(filters, notionapi.DatabaseQueryFilter{
			Property: COLUMN_EFFORT,
			DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
				Select: &notionapi.SelectDatabaseQueryFilter{
					Equals: *args.Effort,
				},
			},
		})
	}

	// Filter by due date
	if args.DueDate != nil && *args.DueDate != "" {
		dueDate, err := time.Parse(DATE_FORMAT, *args.DueDate)
		if err == nil {
			filters = append(filters, notionapi.DatabaseQueryFilter{
				Property: COLUMN_DATE,
				DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
					Date: &notionapi.DatePropertyFilter{
						Equals: &dueDate,
					},
				},
			})
		}
	}

	// Filter by project
	if args.Project != nil && *args.Project != "" {
		filters = append(filters, notionapi.DatabaseQueryFilter{
			Property: COLUMN_PROJECT,
			DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
				Select: &notionapi.SelectDatabaseQueryFilter{
					Equals: *args.Project,
				},
			},
		})
	}

	// Have a default sort by date ascending (earliest tasks first)
	query := notionapi.DatabaseQuery{
		Sorts: []notionapi.DatabaseQuerySort{
			{
				Property:  COLUMN_DATE,
				Direction: notionapi.SortDirAsc,
			},
		},
	}

	// Combine filters with AND if there are any
	if len(filters) > 0 {
		if len(filters) == 1 {
			query.Filter = &filters[0]
		} else {
			query.Filter = &notionapi.DatabaseQueryFilter{
				And: filters,
			}
		}
	}

	return &query
}

// buildUpcomingTasksQuery constructs a Notion database query for upcoming tasks
func (ns *NotionTaskService) buildUpcomingTasksQuery() *notionapi.DatabaseQuery {
	return &notionapi.DatabaseQuery{
		Filter: &notionapi.DatabaseQueryFilter{
			And: []notionapi.DatabaseQueryFilter{
				// 'Complete' is unchecked
				{
					Property: COLUMN_COMPLETE,
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
							Equals: new(false),
						},
					},
				},
				// And 'Canceled' is unchecked
				{
					Property: COLUMN_CANCELED,
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Formula: &notionapi.FormulaDatabaseQueryFilter{
							Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
								Equals: new(false),
							},
						},
					},
				},
				{
					Or: []notionapi.DatabaseQueryFilter{
						// And 'Rank' is greater than five and date is on or before 1 week from now
						{
							Property: COLUMN_RANK,
							DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
								Formula: &notionapi.FormulaDatabaseQueryFilter{
									Number: &notionapi.NumberDatabaseQueryFilter{
										GreaterThanOrEqualTo: new(5),
									},
								},
							},
						},
						// Or 'Date' is on or before one week from now
						{
							Property: COLUMN_DATE,
							DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
								Date: &notionapi.DatePropertyFilter{
									NextWeek: &struct{}{},
								},
							},
						},
						{
							Property: COLUMN_DATE,
							DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
								Date: &notionapi.DatePropertyFilter{
									OnOrBefore: new(time.Now()),
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
				Property:  COLUMN_DATE,
				Direction: notionapi.SortDirAsc,
			},
		},
	}
}

// buildRecurringTasksQuery constructs a Notion database query for active recurring tasks
func (ns *NotionTaskService) buildRecurringTasksQuery() *notionapi.DatabaseQuery {
	return &notionapi.DatabaseQuery{
		Filter: &notionapi.DatabaseQueryFilter{
			And: []notionapi.DatabaseQueryFilter{
				// 'Active' is checked
				{
					Property: RECURRING_COLUMN_ACTIVE,
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
							Equals: new(true),
						},
					},
				},
				// And 'Upcoming' is checked
				{
					Property: RECURRING_COLUMN_UPCOMING,
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Formula: &notionapi.FormulaDatabaseQueryFilter{
							Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
								Equals: new(true),
							},
						},
					},
				},
				// And 'Done' is unchecked
				{
					Property: RECURRING_COLUMN_DONE,
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
							Equals: new(false),
						},
					},
				},
			},
		},
		// Sort by ascending name
		Sorts: []notionapi.DatabaseQuerySort{
			{
				Property:  RECURRING_COLUMN_TITLE,
				Direction: notionapi.SortDirAsc,
			},
		},
	}
}

// buildOverdueQuery constructs a Notion database query for overdue tasks
func (ns *NotionTaskService) buildOverdueTasksQuery() *notionapi.DatabaseQuery {
	return &notionapi.DatabaseQuery{
		Filter: &notionapi.DatabaseQueryFilter{
			And: []notionapi.DatabaseQueryFilter{
				{
					Property: COLUMN_COMPLETE,
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
							Equals: new(false),
						},
					},
				},
				{
					Property: COLUMN_CANCELED,
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
							Equals: new(false),
						},
					},
				},
				{
					Property: COLUMN_DATE,
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Date: &notionapi.DatePropertyFilter{
							Before: new(time.Now()),
						},
					},
				},
			},
		},
		Sorts: []notionapi.DatabaseQuerySort{
			{
				Property:  COLUMN_PRIORITY,
				Direction: notionapi.SortDirDesc,
			},
		},
	}
}

// buildCriticalTasksQuery constructs a Notion database query for critical tasks
func (ns *NotionTaskService) buildCriticalTasksQuery() *notionapi.DatabaseQuery {
	return &notionapi.DatabaseQuery{
		Filter: &notionapi.DatabaseQueryFilter{
			And: []notionapi.DatabaseQueryFilter{
				{
					Property: COLUMN_COMPLETE,
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
							Equals: new(false),
						},
					},
				},
				{
					Property: COLUMN_PRIORITY,
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Select: &notionapi.SelectDatabaseQueryFilter{
							Equals: PRIORITY_CRITICAL,
						},
					},
				},
			},
		},
		Sorts: []notionapi.DatabaseQuerySort{
			{
				Property:  COLUMN_DATE,
				Direction: notionapi.SortDirAsc,
			},
		},
	}
}

// buildUrgentTasksQuery constructs a Notion database query for urgent tasks
func (ns *NotionTaskService) buildUrgentTasksQuery() *notionapi.DatabaseQuery {
	return &notionapi.DatabaseQuery{
		Filter: &notionapi.DatabaseQueryFilter{
			And: []notionapi.DatabaseQueryFilter{
				{
					Property: COLUMN_COMPLETE,
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
							Equals: new(false),
						},
					},
				},
				{
					Or: []notionapi.DatabaseQueryFilter{
						{
							Property: COLUMN_PRIORITY,
							DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
								Select: &notionapi.SelectDatabaseQueryFilter{
									Equals: PRIORITY_HIGH,
								},
							},
						},
						{
							Property: COLUMN_PRIORITY,
							DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
								Select: &notionapi.SelectDatabaseQueryFilter{
									Equals: PRIORITY_CRITICAL,
								},
							},
						},
					},
				},
				{
					Property: COLUMN_DATE,
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Date: &notionapi.DatePropertyFilter{
							NextWeek: &struct{}{},
						},
					},
				},
			},
		},
		Sorts: []notionapi.DatabaseQuerySort{
			{
				Property:  COLUMN_PRIORITY,
				Direction: notionapi.SortDirDesc,
			},
			{
				Property:  COLUMN_DATE,
				Direction: notionapi.SortDirAsc,
			},
		},
	}
}

// buildQuickWinsQuery constructs a Notion database query for quick win tasks
func (ns *NotionTaskService) buildQuickWinsQuery() *notionapi.DatabaseQuery {
	return &notionapi.DatabaseQuery{
		Filter: &notionapi.DatabaseQueryFilter{
			And: []notionapi.DatabaseQueryFilter{
				{
					Property: COLUMN_COMPLETE,
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
							Equals: new(false),
						},
					},
				},
				{
					Property: COLUMN_EFFORT,
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Select: &notionapi.SelectDatabaseQueryFilter{
							Equals: EFFORT_LOW,
						},
					},
				},
			},
		},
		Sorts: []notionapi.DatabaseQuerySort{
			{
				Property:  COLUMN_DATE,
				Direction: notionapi.SortDirAsc,
			},
			{
				Property:  COLUMN_PRIORITY,
				Direction: notionapi.SortDirDesc,
			},
		},
	}
}

func (ns *NotionTaskService) buildScheduleItemsQuery() *notionapi.DatabaseQuery {
	now := time.Now().In(ns.tz)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, ns.tz)
	tomorrow := today.Add(24 * time.Hour)

	return &notionapi.DatabaseQuery{
		Filter: &notionapi.DatabaseQueryFilter{
			And: []notionapi.DatabaseQueryFilter{
				{
					Property: "Date",
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Date: &notionapi.DatePropertyFilter{
							OnOrAfter: &today,
						},
					},
				},
				{
					Property: "Date",
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Date: &notionapi.DatePropertyFilter{
							OnOrBefore: &tomorrow,
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
	}
}
