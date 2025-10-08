package outreach_notionschedule

import (
	"bytes"
	"fmt"
	"net/http"
	"sync"
	"time"

	notionapi "github.com/dstotijn/go-notion"
	"github.com/ethanbaker/assistant/pkg/utils"
)

/* ---- GLOBALS ---- */

// Notion client
var notion *notionapi.Client

// Global list of events protected by mutex
var events []Event
var eventsMutex sync.RWMutex

/* ---- INIT ---- */

func Init(cfg *utils.Config) error {
	token := cfg.Get("NOTION_API_TOKEN")
	if token == "" {
		return fmt.Errorf("NOTION_API_TOKEN environment variable is not set")
	}

	// Initialize Notion client
	notion = notionapi.NewClient(token, notionapi.WithHTTPClient(&http.Client{
		Timeout:   20 * time.Second,
		Transport: &httpTransport{w: &bytes.Buffer{}},
	}))

	// Update notion queries
	SCHEDULE_ITEMS.ID = cfg.Get("NOTION_DATABASE_SCHEDULE_ID")
	if SCHEDULE_ITEMS.ID == "" {
		return fmt.Errorf("NOTION_DATABASE_SCHEDULE_ID environment variable is not set")
	}

	// Start the background goroutine to fetch events
	go func() {
		for {
			// Fetch events and update the global list
			fetchNotionEvents(cfg)

			// Wait for the specified interval
			time.Sleep(UPDATE_INTERVAL_MINUTES * time.Minute)
		}
	}()

	return nil
}
