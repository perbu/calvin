package dateparse

import (
	"errors"
	"log"
	"time"
)

// Parser defines the interface for parsing dates.
type Parser interface {
	Parse(args []string) (string, time.Time, error)
}

// DefaultParser implements the Parser interface.
type DefaultParser struct{}

// Parse parses command-line arguments to extract username and date.
func (p *DefaultParser) Parse(args []string) (string, time.Time, error) {
	if len(args) == 0 {
		return "", time.Time{}, errors.New("missing username")
	}
	if len(args) > 2 {
		return "", time.Time{}, errors.New("too many arguments")
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
