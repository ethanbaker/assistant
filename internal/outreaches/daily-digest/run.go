package outreach_dailydigest

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
	notionapi "github.com/dstotijn/go-notion"
	"github.com/ethanbaker/assistant/pkg/outreach"
	"github.com/ethanbaker/assistant/pkg/utils"
	"github.com/teambition/rrule-go"
)

/* ---- METHODS ---- */

func CreateDailyDigest(cfg *utils.Config) *outreach.TaskReturn {
	var output string
	var err error

	// Run generator for a given number of times
	for range RETRY_LIMIT {
		output, err = getDailyDigest(cfg)

		// On no error, return the output
		if err == nil {
			return &outreach.TaskReturn{
				Content: output,
				Data:    nil,
			}
		}
		log.Printf("[DAILY-DIGEST]: Error in getting Daily Digest, retrying (err: %v)\n", err)
	}

	// Return the last error if we only fail
	return &outreach.TaskReturn{
		Content: fmt.Sprintf("Error getting daily digest<BLOCKQUOTE>Error: %v</BLOCKQUOTE>", err),
		Data:    nil,
	}
}

// helper function to get the daily digest with associated errors
func getDailyDigest(_ *utils.Config) (string, error) {
	var output string

	var events []Event

	// Get calendar events
	calendarEvents := getCalendarEvents()
	events = append(events, calendarEvents...)

	// Get schedule database events
	scheduleEvents, err := getScheduleDatabaseEvents()
	if err != nil {
		return "", err
	}
	events = append(events, scheduleEvents...)

	// Sort events
	for i := 1; i < len(events); i++ {
		event := events[i]
		j := i - 1

		for j >= 0 && events[j].Start.Compare(event.Start) > 0 {
			events[j+1] = events[j]
			j--
		}
		events[j+1] = event
	}

	// Add calendar events to the output
	if len(events) != 0 {
		output += "<STRONG>Schedule:</STRONG>\n"
	}

	for _, event := range events {
		output += fmt.Sprintf("- %v: %v\n", event.Timespan, event.Title)
	}

	// Add upcoming tasks content
	upcomingTasks, err := getUpcomingTasks()
	if err != nil {
		return "", err
	}
	output += upcomingTasks

	// Add critical tasks content
	criticalTasks, err := getCriticalTasks()
	if err != nil {
		return "", err
	}
	output += criticalTasks

	// Add recurring tasks content
	recurringTasks, err := getRecurringTasks()
	if err != nil {
		return "", err
	}
	output += recurringTasks

	return output, nil
}

// Helper method to get events from the calendars
func getCalendarEvents() []Event {
	calendarEvents := []Event{}

	// Find 'today' in the calendar's timezone (assuming there is only one timezone)
	year, month, day := time.Now().In(formatLoc).Date()
	today := time.Date(year, month, day, 0, 0, 0, 0, formatLoc)

	// Loop through each calendar event
	for _, calendar := range calendars {
		for _, event := range calendar.Events() {
			// Get the name
			name := event.GetProperty(ics.ComponentPropertySummary)

			// Get all times of the event
			start, _ := event.GetStartAt()
			end, endNotPresent := event.GetEndAt()
			startDay, _ := event.GetAllDayStartAt()
			endDay, endDayNotPresent := event.GetAllDayEndAt()

			// Convert to local timezone
			start = start.In(formatLoc)
			end = end.In(formatLoc)
			startDay = startDay.In(formatLoc)
			endDay = endDay.In(formatLoc)

			// Determine whether this is an all day event
			allDayEvent := start.Equal(startDay) && end.Equal(endDay)
			allDayEvent = allDayEvent || endNotPresent != nil && endDayNotPresent != nil // Weird hack for events without end saved -> are all day events

			// Edit the start date if there are repeat rules
			rr := event.GetProperty(ics.ComponentPropertyRrule)
			repeating := rr != nil

			if repeating {
				// Get the recurring rule
				rule, err := rrule.StrToRRule(rr.BaseProperty.Value)
				if err != nil {
					continue // Skip on error, assume malformatted data
				}

				if allDayEvent {
					// For all day events, set to the base date of start
					year, month, day := start.Date()
					rule.DTStart(time.Date(year, month, day, 0, 0, 0, 0, formatLoc))
				} else {
					// For normal events, just set to start
					rule.DTStart(start)
				}

				// Calculate the next occurence
				start = rule.After(today, true)
				startDay = start
				endDay = startDay.Add(24 * time.Hour)

				// Check if the next occurence is excluded
				for _, prop := range event.Properties {
					if prop.IANAToken == "EXDATE" {
						// Get the time from the EXDATE
						t, err := time.Parse("20060102T150405", prop.Value)
						if err != nil {
							continue
						}
						t = t.In(formatLoc)

						// If time is equal to start time reject
						if start.Truncate(24 * time.Hour).Equal(t.Truncate(24 * time.Hour)) {
							start = time.Time{}
							break
						}
					}
				}
			}

			// Set flags
			isBusy := strings.Contains(strings.ToLower(name.BaseProperty.Value), "busy")

			if allDayEvent {
				// Add an all day event
				if (today.After(startDay) || today.Equal(startDay)) && today.Before(endDay) {
					calendarEvents = append(calendarEvents, Event{
						Start:    start,
						Title:    name.BaseProperty.Value,
						Timespan: "All Day",
						IsBusy:   isBusy,
						IsAllDay: true,
					})
				}
			} else {
				// Normal days have no errors from end days or end times
				if today.Day() == start.Day() && today.Month() == start.Month() && today.Year() == start.Year() {
					calendarEvents = append(calendarEvents, Event{
						Start:    start,
						Title:    name.BaseProperty.Value,
						Timespan: fmt.Sprintf("%v → %v", start.In(formatLoc).Format("3:04 PM"), end.In(formatLoc).Format("3:04 PM")),
						IsBusy:   isBusy,
						IsAllDay: false,
					})
				}
			}
		}
	}

	// Remove duplicate "busy" events (name = "Busy", same time as another event)
	uniqueEvents := []Event{}
	for _, event := range calendarEvents {
		// If not busy, always add
		if !event.IsBusy {
			uniqueEvents = append(uniqueEvents, event)
			continue
		}

		// Check if there is another event at the same time
		duplicate := false
		for _, other := range calendarEvents {
			if !other.IsBusy && event.Timespan == other.Timespan {
				// Matching non-busy event found, do not add
				duplicate = true
				break
			}
		}
		if !duplicate {
			uniqueEvents = append(uniqueEvents, event)
		}
	}

	return uniqueEvents
}

// Helper function to get events from the schedule database
func getScheduleDatabaseEvents() ([]Event, error) {
	var scheduleEvents []Event

	// Find "today"
	year, month, day := time.Now().In(formatLoc).Date()
	today := time.Date(year, month, day, 0, 0, 0, 0, formatLoc)

	// Get the schedule database
	schedule, err := notion.QueryDatabase(context.Background(), SCHEDULE_ITEMS.ID, &SCHEDULE_ITEMS.Query)
	if err != nil {
		return nil, err
	}

	// Loop for each task page
	for _, p := range schedule.Results {
		// Get the page property IDs from Notion
		page, err := notion.FindPageByID(context.Background(), p.ID)
		if err != nil {
			return nil, err
		}

		// Get the page property values from their IDs
		properties, ok := page.Properties.(notionapi.DatabasePageProperties)
		if !ok {
			return nil, err
		}

		// Get the name of the task
		nameField := properties["Name"]
		if len(nameField.Title) == 0 {
			continue
		}
		name := nameField.Title[0].Text.Content

		// Get the start and end of the task
		startField := properties["Date"]
		start := startField.Date.Start.Time

		// Only add if the start date is today
		if start.Day() != today.Day() {
			continue
		}

		// Get the end time if it exists. If end time doesn't exist, it is an all day event
		var end *time.Time
		if startField.Date.End != nil {
			end = &startField.Date.End.Time
		}

		// Format timespan string
		var timespan string
		if end == nil {
			timespan = "All Day"
		} else {
			timespan = fmt.Sprintf("%v → %v", start.Format("3:04PM"), end.Format("3:04PM"))
		}

		// Add the event to the calendar events
		scheduleEvents = append(scheduleEvents, Event{
			Start:    start,
			Title:    name,
			Timespan: timespan,
			IsBusy:   false,
			IsAllDay: end == nil,
		})
	}

	return scheduleEvents, nil
}

// Helper method to add upcoming tasks
func getUpcomingTasks() (string, error) {
	var output string

	// Get the tasks page
	tasks, err := notion.QueryDatabase(context.Background(), NORMAL_TASKS.ID, &NORMAL_TASKS.Query)
	if err != nil {
		return "", err
	}

	if len(tasks.Results) != 0 {
		output += "\n<STRONG>Upcoming Tasks:</STRONG>\n"
	}

	// Loop for each task page
	for _, p := range tasks.Results {
		// Get the page property IDs from Notion
		page, err := notion.FindPageByID(context.Background(), p.ID)
		if err != nil {
			return "", err
		}

		// Get the page property values from their IDs
		properties, ok := page.Properties.(notionapi.DatabasePageProperties)
		if !ok {
			return "", err
		}

		// Get the name of the task
		nameField := properties["Name"]
		if len(nameField.Title) == 0 {
			continue
		}
		name := nameField.Title[0].Text.Content

		// Get the project of the task
		projectField := properties["Project Name"]

		project := ""
		if projectField.Formula != nil && projectField.Formula.String != nil {
			project = *projectField.Formula.String
		}

		if project != "" {
			project = "<EM>" + project + "</EM>"
		}

		// Get the date of the task
		dateField := properties["Date"]
		date := ""
		if dateField.Date != nil {
			date = "(" + dateField.Date.Start.Format("Mon Jan 2") + ")"
		}

		output += fmt.Sprintf("- %v %v %v\n", name, project, date)
	}

	return output, nil
}

// Helper method to get critical tasks
func getCriticalTasks() (string, error) {
	var output string

	// Get the tasks page
	criticalTasks, err := notion.QueryDatabase(context.Background(), CRITICAL_TASKS.ID, &CRITICAL_TASKS.Query)
	if err != nil {
		return "", err
	}

	if len(criticalTasks.Results) != 0 {
		output += "\n<STRONG>Critical Tasks:</STRONG>\n"
	}

	// Loop for each task page
	for _, p := range criticalTasks.Results {
		// Get the page property IDs from Notion
		page, err := notion.FindPageByID(context.Background(), p.ID)
		if err != nil {
			return "", err
		}

		// Get the page property values from their IDs
		properties, ok := page.Properties.(notionapi.DatabasePageProperties)
		if !ok {
			return "", err
		}

		// Get the name of the task
		nameField := properties["Name"]
		if len(nameField.Title) == 0 {
			continue
		}
		name := nameField.Title[0].Text.Content

		// Get the project of the task
		projectField := properties["Tasks -> Project Name"]

		project := ""
		if projectField.Formula != nil && projectField.Formula.String != nil {
			project = *projectField.Formula.String
		}

		if project != "" {
			project = "<EM>" + project + "</EM>"
		}

		// Get the date of the task
		dateField := properties["Date"]
		date := ""
		if dateField.Date != nil {
			date = "(" + dateField.Date.Start.Format("Mon Jan 2") + ")"
		}

		output += fmt.Sprintf("- %v %v %v\n", name, project, date)
	}

	return output, nil
}

func getRecurringTasks() (string, error) {
	var output string

	// Get the recurring database sections
	for _, t := range []string{"Connection", "Habit", "Chore"} {
		RECURRING_TASKS.Query.Filter.And[3].Select.Equals = t

		recurring, err := notion.QueryDatabase(context.Background(), RECURRING_TASKS.ID, &RECURRING_TASKS.Query)
		if err != nil {
			return "", err
		}

		if len(recurring.Results) != 0 {
			output += fmt.Sprintf("\n<STRONG>%vs:</STRONG>\n", t)
		}

		// Loop for each task page
		for _, p := range recurring.Results {
			// Get the page property IDs from Notion
			page, err := notion.FindPageByID(context.Background(), p.ID)
			if err != nil {
				return "", err
			}

			// Get the page property values from their IDs
			properties, ok := page.Properties.(notionapi.DatabasePageProperties)
			if !ok {
				return "", err
			}

			// Get the name of the task
			nameField := properties["Name"]
			if len(nameField.Title) == 0 {
				continue
			}
			name := nameField.Title[0].Text.Content

			output += fmt.Sprintf("- %v\n", name)
		}
	}

	return output, nil
}
