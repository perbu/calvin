package gcal

import (
	"time"

	"google.golang.org/api/calendar/v3"
)

// CalendarService defines the interface for interacting with Google Calendar.
type CalendarService interface {
	ListEvents(calendarID string, theDate time.Time) (*calendar.Events, error)
}
