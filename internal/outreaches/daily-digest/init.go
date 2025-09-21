package outreach_dailydigest

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"time"
	_ "time/tzdata" // Embed timezone database

	ics "github.com/arran4/golang-ical"
	notionapi "github.com/dstotijn/go-notion"
	"github.com/ethanbaker/assistant/pkg/utils"
	"gopkg.in/yaml.v3"
)

/* ---- GLOBALS ---- */

// Calendars for parsing iCal formats
var calendars []*ics.Calendar

// Preferred timezone for formatting
var formatLoc *time.Location

// Notion client
var notion *notionapi.Client

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
	NORMAL_TASKS.ID = cfg.Get("NOTION_DATABASE_TASKS_ID")
	if NORMAL_TASKS.ID == "" {
		return fmt.Errorf("NOTION_DATABASE_TASKS_ID environment variable is not set")
	}

	CRITICAL_TASKS.ID = cfg.Get("NOTION_DATABASE_TASKS_ID")
	if CRITICAL_TASKS.ID == "" {
		return fmt.Errorf("NOTION_DATABASE_TASKS_ID environment variable is not set")
	}

	SCHEDULE_ITEMS.ID = cfg.Get("NOTION_DATABASE_SCHEDULE_ID")
	if SCHEDULE_ITEMS.ID == "" {
		return fmt.Errorf("NOTION_DATABASE_SCHEDULE_ID environment variable is not set")
	}

	RECURRING_TASKS.ID = cfg.Get("NOTION_DATABASE_RECURRING_ID")
	if RECURRING_TASKS.ID == "" {
		return fmt.Errorf("NOTION_DATABASE_RECURRING_ID environment variable is not set")
	}

	// Read in calendar config
	calendarPath := cfg.Get("CALENDAR_CONFIG_FILE")
	if calendarPath == "" {
		return fmt.Errorf("CALENDAR_CONFIG_FILE environment variable is not set")
	}

	yamlFile, err := os.ReadFile(calendarPath)
	if err != nil {
		return err
	}

	var config CalendarConfig
	if err = yaml.Unmarshal(yamlFile, &config); err != nil {
		return err
	}

	// Repeat for each calendar
	for _, cal := range config.Calendars {
		// Get the URL
		resp, err := http.Get(cal.URL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// Create the calendar
		c, err := ics.ParseCalendar(resp.Body)
		if err != nil {
			return err
		}

		calendars = append(calendars, c)
	}

	// Load the format location
	formatLoc, err = time.LoadLocation(config.TimezoneFormat)
	if err != nil {
		return err
	}

	return nil
}
