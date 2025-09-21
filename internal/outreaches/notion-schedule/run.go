package outreach_notionschedule

import (
	"fmt"
	"time"

	"github.com/ethanbaker/assistant/pkg/outreach"
	"github.com/ethanbaker/assistant/pkg/utils"
)

/* ---- OUTREACH TASK ---- */

// NotionScheduleReminder checks if there's an event occurring in the current minute
func NotionScheduleReminder(cfg *utils.Config) *outreach.TaskReturn {
	now := time.Now()

	// Get the current list of events (thread-safe)
	eventsMutex.RLock()
	currentEvents := make([]Event, len(events))
	copy(currentEvents, events)
	eventsMutex.RUnlock()

	// Find events that start in the current minute
	var activeEvents []Event
	var remainingEvents []Event
	for _, e := range currentEvents {
		// If the event is this minute, add it
		if now.Truncate(time.Minute).Equal(e.Start.Truncate(time.Minute)) {
			activeEvents = append(activeEvents, e)
		} else {
			remainingEvents = append(remainingEvents, e)
		}
	}

	// Do formatting and return output
	if len(activeEvents) == 0 {
		return nil
	} else if len(activeEvents) == 1 {
		return &outreach.TaskReturn{
			Content: fmt.Sprintf("<STRONG>Schedule Event:</STRONG> %v (%v)\n", activeEvents[0].Name, activeEvents[0].Timespan),
			Data:    nil,
		}
	}

	// Handle the multi event case
	output := "<STRONG>Schedule Events:</STRONG>\n"
	for _, e := range activeEvents {
		output += fmt.Sprintf("- %v (%v)\n", e.Name, e.Timespan)
	}

	// Remove events that just occurred
	eventsMutex.Lock()
	events = remainingEvents
	eventsMutex.Unlock()

	return &outreach.TaskReturn{
		Content: output,
		Data:    nil,
	}
}
