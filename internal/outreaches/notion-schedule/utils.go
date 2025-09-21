package outreach_notionschedule

import (
	"context"
	"fmt"
	"log"

	notionapi "github.com/dstotijn/go-notion"
	"github.com/ethanbaker/assistant/pkg/utils"
)

/* ---- BACKGROUND UPDATE FUNCTION ---- */

// fetchNotionEvents fetches events from Notion and updates the global events list
func fetchNotionEvents(_ *utils.Config) {
	var newEvents []Event
	var mainErr error

	// Run for a given number of times for retry logic
	for range SCHEDULE_REMINDERS_ERROR_LIMIT {
		// Query the schedule items
		schedule, err := notion.QueryDatabase(context.Background(), SCHEDULE_ITEMS.ID, &SCHEDULE_ITEMS.Query)
		if err != nil {
			log.Printf("[NOTION-SCHEDULE]: Error in fetchNotionEvents, retrying (err: %v)\n", err)
			continue
		}

		// Loop for each task page
		for _, p := range schedule.Results {
			// Get the page property IDs from Notion
			page, err := notion.FindPageByID(context.Background(), p.ID)
			if err != nil {
				mainErr = err
				break
			}

			// Get the page property values from their IDs
			properties, ok := page.Properties.(notionapi.DatabasePageProperties)
			if !ok {
				mainErr = fmt.Errorf("cannot cast properties")
				break
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
			end := startField.Date.End.Time

			// Format timespan string
			timespan := fmt.Sprintf("%v - %v", start.Format("3:04PM"), end.Format("3:04PM"))

			// Add the event to the calendar events
			newEvents = append(newEvents, Event{
				Start:    start,
				Name:     name,
				Timespan: timespan,
			})
		}

		// On no error, break
		if mainErr == nil {
			break
		}
		log.Printf("[NOTION-SCHEDULE]: Error in fetchNotionEvents, retrying (err: %v)\n", mainErr)
	}

	// If there's no error, update the global events list
	if mainErr == nil {
		eventsMutex.Lock()
		events = newEvents
		eventsMutex.Unlock()
	} else {
		log.Printf("[NOTION-SCHEDULE]: Failed to fetch events after retries: %v\n", mainErr)
	}
}
