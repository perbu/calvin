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
	NowDate func(string) time.Time // NowDate is a function that returns the current date as time.Time
}

func New() *DefaultParser {
	return &DefaultParser{
		NowDate: func(layout string) time.Time {
			return time.Now().Truncate(24 * time.Hour)
		},
	}
}

// Parse parses command-line arguments to extract username and date.
func (p *DefaultParser) Parse(args []string) (string, time.Time, error) {
	if len(args) == 0 {
		return "", time.Time{}, errors.New("missing username")
	}
	username := args[0]
	theDate := time.Now().Truncate(24 * time.Hour)
	if len(args) == 1 {
		return username, theDate, nil
	}

	switch args[1] {
	case "":
		// keep today's date
	case "tomorrow":
		theDate = theDate.Add(24 * time.Hour)
	case "next": // next monday, next tuesday, etc.
		if len(args) < 3 {
			return "", time.Time{}, errors.New("missing day of week")
		}
		weekday := strings.ToLower(args[2])
		for i := 0; i < 7; i++ { //
			if strings.ToLower(theDate.Weekday().String()) == weekday {
				return username, theDate, nil
			}
			theDate = theDate.Add(24 * time.Hour)
		}
		return "", time.Time{}, fmt.Errorf("invalid day of week: %s", args[2])
	default:
		parsed, err := time.Parse("2006-01-02", args[1])
		if err == nil {
			theDate = parsed
		} else {
			log.Printf("Warning: could not parse date %q, using today", args[1])
		}
	}
	return username, theDate, nil
}
