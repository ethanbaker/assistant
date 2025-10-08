package outreach_dailydigest

import (
	"io"
	"net/http"
	"time"

	notionapi "github.com/dstotijn/go-notion"
)

/* ---- TYPES ---- */

// NotionDatabase type holds a database ID and query to get the database with
type NotionDatabase struct {
	ID    string
	Query notionapi.DatabaseQuery
}

// Event type used to hold calendar events for nice formatting
type Event struct {
	Start    time.Time
	Title    string
	Timespan string
	IsBusy   bool
	IsAllDay bool
}

// CalendarConfig holds the configuration for calendar integration
type CalendarConfig struct {
	Calendars []struct {
		URL string `yaml:"url"`
	} `yaml:"calendars"`
	TimezoneFormat string `yaml:"timezone-format"`
}

type httpTransport struct {
	w io.Writer
}

// RoundTrip implements http.RoundTripper. It multiplexes the read HTTP response
// data to an io.Writer for debugging.
func (t *httpTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	res.Body = io.NopCloser(io.TeeReader(res.Body, t.w))

	return res, nil
}
