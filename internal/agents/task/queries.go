package task

import (
	"time"

	notionapi "github.com/dstotijn/go-notion"
)

func (ta *TaskAgent) buildFetchTasksQuery(args FetchTasksArgs) notionapi.DatabaseQuery {
	filters := []notionapi.DatabaseQueryFilter{}

	// Always filter canceled tasks
	filters = append(filters, notionapi.DatabaseQueryFilter{
		Property: COLUMN_CANCELED,
		DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
			Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
				Equals: pointer(false),
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

	return query
}

// buildUpcomingTasksQuery constructs a Notion database query for upcoming tasks
func (ta *TaskAgent) buildUpcomingTasksQuery() notionapi.DatabaseQuery {
	return notionapi.DatabaseQuery{
		Filter: &notionapi.DatabaseQueryFilter{
			And: []notionapi.DatabaseQueryFilter{
				// 'Complete' is unchecked
				{
					Property: COLUMN_COMPLETE,
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
							Equals: pointer(false),
						},
					},
				},
				// And 'Canceled' is unchecked
				{
					Property: COLUMN_CANCELED,
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Formula: &notionapi.FormulaDatabaseQueryFilter{
							Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
								Equals: pointer(false),
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
										GreaterThanOrEqualTo: pointer(5),
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
									OnOrBefore: pointer(time.Now()),
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
func (ta *TaskAgent) buildRecurringTasksQuery() notionapi.DatabaseQuery {
	return notionapi.DatabaseQuery{
		Filter: &notionapi.DatabaseQueryFilter{
			And: []notionapi.DatabaseQueryFilter{
				// 'Active' is checked
				{
					Property: RECURRING_COLUMN_ACTIVE,
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
							Equals: pointer(true),
						},
					},
				},
				// And 'Upcoming' is checked
				{
					Property: RECURRING_COLUMN_UPCOMING,
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Formula: &notionapi.FormulaDatabaseQueryFilter{
							Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
								Equals: pointer(true),
							},
						},
					},
				},
				// And 'Done' is unchecked
				{
					Property: RECURRING_COLUMN_DONE,
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
							Equals: pointer(false),
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
func (ta *TaskAgent) buildOverdueTasksQuery() notionapi.DatabaseQuery {
	return notionapi.DatabaseQuery{
		Filter: &notionapi.DatabaseQueryFilter{
			And: []notionapi.DatabaseQueryFilter{
				{
					Property: COLUMN_COMPLETE,
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
							Equals: pointer(false),
						},
					},
				},
				{
					Property: COLUMN_CANCELED,
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
							Equals: pointer(false),
						},
					},
				},
				{
					Property: COLUMN_DATE,
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Date: &notionapi.DatePropertyFilter{
							Before: pointer(time.Now()),
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
func (ta *TaskAgent) buildCriticalTasksQuery() notionapi.DatabaseQuery {
	return notionapi.DatabaseQuery{
		Filter: &notionapi.DatabaseQueryFilter{
			And: []notionapi.DatabaseQueryFilter{
				{
					Property: COLUMN_COMPLETE,
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
							Equals: pointer(false),
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
func (ta *TaskAgent) buildUrgentTasksQuery() notionapi.DatabaseQuery {
	return notionapi.DatabaseQuery{
		Filter: &notionapi.DatabaseQueryFilter{
			And: []notionapi.DatabaseQueryFilter{
				{
					Property: COLUMN_COMPLETE,
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
							Equals: pointer(false),
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
func (ta *TaskAgent) buildQuickWinsQuery() notionapi.DatabaseQuery {
	return notionapi.DatabaseQuery{
		Filter: &notionapi.DatabaseQueryFilter{
			And: []notionapi.DatabaseQueryFilter{
				{
					Property: COLUMN_COMPLETE,
					DatabaseQueryPropertyFilter: notionapi.DatabaseQueryPropertyFilter{
						Checkbox: &notionapi.CheckboxDatabaseQueryFilter{
							Equals: pointer(false),
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
