package gcal

import (
	"testing"
	"time"

	"google.golang.org/api/calendar/v3"
)

// MockCalendarService is a mock implementation of CalendarService.
type MockCalendarService struct {
	Events *calendar.Events
	Err    error
}

func (m *MockCalendarService) ListEvents(calendarID string, theDate time.Time) (*calendar.Events, error) {
	return m.Events, m.Err
}

func TestListAndPrintEvents(t *testing.T) {
	mockEvents := &calendar.Events{
		Items: []*calendar.Event{
			{
				Summary: "Meeting with Bob",
				Start: &calendar.EventDateTime{
					DateTime: "2025-01-31T10:00:00-07:00",
				},
				End: &calendar.EventDateTime{
					DateTime: "2025-01-31T11:00:00-07:00",
				},
			},
			{
				Summary: "Lunch",
				Start: &calendar.EventDateTime{
					Date: "2025-01-31",
				},
			},
		},
	}

	mockService := &MockCalendarService{
		Events: mockEvents,
		Err:    nil,
	}

	err := ListAndPrintEvents(mockService, "alice@example.com", time.Date(2025, 1, 31, 0, 0, 0, 0, time.Local), "example.com", nil)
	if err != nil {
		t.Errorf("ListAndPrintEvents returned error: %v", err)
	}
}

func TestListAndPrintEventsForWeek(t *testing.T) {
	mockEvents := &calendar.Events{
		Items: []*calendar.Event{
			{
				Summary: "Meeting with Bob",
				Start: &calendar.EventDateTime{
					DateTime: "2025-01-31T10:00:00-07:00",
				},
				End: &calendar.EventDateTime{
					DateTime: "2025-01-31T11:00:00-07:00",
				},
			},
		},
	}

	mockService := &MockCalendarService{
		Events: mockEvents,
		Err:    nil,
	}

	// Create a week of dates (Monday to Sunday)
	weekDays := make([]time.Time, 7)
	monday := time.Date(2025, 1, 27, 0, 0, 0, 0, time.Local) // Monday
	for i := 0; i < 7; i++ {
		weekDays[i] = monday.AddDate(0, 0, i)
	}

	err := ListAndPrintEventsForWeek(mockService, "alice@example.com", weekDays, "example.com", nil)
	if err != nil {
		t.Errorf("ListAndPrintEventsForWeek returned error: %v", err)
	}
}
