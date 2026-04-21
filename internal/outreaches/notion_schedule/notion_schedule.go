package notionschedule

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ethanbaker/assistant/internal/api/modules/outreach"
	"github.com/ethanbaker/assistant/internal/services/notion"
)

const updateInterval = 5 * time.Minute

// NotionSchedule polls Notion for today's schedule events and surfaces any
// event that starts in the current minute when RunNotionSchedule is called.
type NotionSchedule struct {
	notionService notion.TaskService
	mu            sync.RWMutex
	events        []notion.ScheduleItem
}

// NewNotionSchedule creates a NotionSchedule and starts a background goroutine
// that refreshes the event list from Notion every updateInterval. The goroutine
// stops when ctx is cancelled.
func NewNotionSchedule(ctx context.Context, notionService notion.TaskService) *NotionSchedule {
	ns := &NotionSchedule{notionService: notionService}
	ns.fetchEvents(ctx)
	go ns.runFetcher(ctx)
	return ns
}

func (ns *NotionSchedule) runFetcher(ctx context.Context) {
	ticker := time.NewTicker(updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ns.fetchEvents(ctx)
		}
	}
}

func (ns *NotionSchedule) fetchEvents(ctx context.Context) {
	events, err := ns.notionService.QueryScheduleItems(ctx)
	if err != nil {
		log.Printf("[NOTION-SCHEDULE]: failed to fetch events: %v\n", err)
		return
	}

	ns.mu.Lock()
	ns.events = events
	ns.mu.Unlock()
}

// RunNotionSchedule checks whether any cached schedule event starts in the
// current minute. Matching events are removed from the cache and returned as a
// formatted string. Returns ("", nil) when nothing is starting right now.
func (ns *NotionSchedule) RunNotionSchedule(_ context.Context, _ json.RawMessage) (*outreach.OutreachResponse, error) {
	now := time.Now().Truncate(time.Minute)

	ns.mu.Lock()
	defer ns.mu.Unlock()

	var active, remaining []notion.ScheduleItem
	for _, e := range ns.events {
		if e.Start != nil && e.Start.Truncate(time.Minute).Equal(now) {
			active = append(active, e)
		} else {
			remaining = append(remaining, e)
		}
	}

	if len(active) == 0 {
		return nil, nil
	}

	ns.events = remaining

	if len(active) == 1 {
		return &outreach.OutreachResponse{
			Content: fmt.Sprintf("<STRONG>Schedule Event:</STRONG> %s (%s)\n", active[0].Title, safeTimespan(active[0])),
		}, nil
	}

	output := "<STRONG>Schedule Events:</STRONG>\n"
	for _, e := range active {
		output += fmt.Sprintf("- %s (%s)\n", e.Title, safeTimespan(e))
	}
	return &outreach.OutreachResponse{
		Content: output,
	}, nil
}

func safeTimespan(e notion.ScheduleItem) string {
	if e.Timespan != nil {
		return *e.Timespan
	}
	return ""
}
