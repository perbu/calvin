package dateparse

import (
	"errors"
	"fmt"
	_ "golang.org/x/text/cases"
	"log"
	"strings"
	"time"
)

// Parser defines the interface for parsing dates.
type Parser interface {
	Parse(args []string) (string, time.Time, error)
}

// DefaultParser implements the Parser interface.
type DefaultParser struct {
	NowDate func() time.Time // NowDate is a function that returns the current date as time.Time
}

func New() *DefaultParser {
	return &DefaultParser{
		NowDate: func() time.Time {
			return time.Now().Truncate(24 * time.Hour)
		},
	}
}

// ParseResult contains the result of parsing date arguments
type ParseResult struct {
	Date     time.Time
	IsWeek   bool
	WeekDays []time.Time
}

// Parse parses command-line arguments to extract username and date.
func (p *DefaultParser) Parse(args []string) (ParseResult, error) {
	result := ParseResult{
		Date:   p.NowDate().Truncate(24 * time.Hour),
		IsWeek: false,
	}

	if len(args) <= 1 {
		return result, nil
	}

	switch args[1] {
	case "":
		// keep today's date
	case "today":
		// keep today's date
	case "tomorrow":
		result.Date = result.Date.Add(24 * time.Hour)
	case "yesterday":
		result.Date = result.Date.Add(-24 * time.Hour)
	case "week":
		// Get the current week (starting from today)
		result.IsWeek = true
		result.WeekDays = getWeekDays(result.Date, 0)
	case "next":
		if len(args) < 3 {
			return ParseResult{}, errors.New("missing day of week or 'week'")
		}

		if strings.ToLower(args[2]) == "week" {
			// Get next week (starting from next Monday)
			result.IsWeek = true
			result.WeekDays = getWeekDays(result.Date, 7)
			return result, nil
		}

		// Handle "next monday", "next tuesday", etc.
		weekday := strings.ToLower(args[2])
		for i := 0; i < 7; i++ {
			if strings.ToLower(result.Date.Weekday().String()) == weekday {
				return result, nil
			}
			result.Date = result.Date.Add(24 * time.Hour)
		}
		return ParseResult{}, fmt.Errorf("invalid day of week: %s", args[2])
	default:
		parsed, err := time.Parse("2006-01-02", args[1])
		if err == nil {
			result.Date = parsed
		} else {
			log.Printf("Warning: could not parse date %q, using today", args[1])
		}
	}
	return result, nil
}

// getWeekDays returns an array of time.Time objects representing days in a week
// offset is the number of days to add to the start date before calculating the week
func getWeekDays(startDate time.Time, offset int) []time.Time {
	// Add the offset to get to the desired week
	startDate = startDate.AddDate(0, 0, offset)

	// Find the Monday of the week
	daysUntilMonday := int(time.Monday - startDate.Weekday())
	if daysUntilMonday > 0 {
		daysUntilMonday -= 7 // Adjust if we're already past Monday
	}

	monday := startDate.AddDate(0, 0, daysUntilMonday)

	// Create an array of 7 days starting from Monday
	weekDays := make([]time.Time, 7)
	for i := 0; i < 7; i++ {
		weekDays[i] = monday.AddDate(0, 0, i)
	}

	return weekDays
}
