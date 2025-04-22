package main

import (
	_ "embed"
	"flag"
	"fmt"
	"github.com/perbu/calvin/config"
	"github.com/perbu/calvin/dateparse"
	"github.com/perbu/calvin/gcal"
	"log"
	"os"
	"time"
)

//go:embed .version
var embeddedVersion string

func run(args []string) error {
	// Initialize configuration loader

	var useLocalTimezone bool

	flag.BoolVar(&useLocalTimezone, "local", false, "Use local timezone")
	flag.Parse()

	loader, err := config.NewFileLoader()
	if err != nil {
		return fmt.Errorf("config.NewFileLoader: %w", err)
	}

	// Load configuration
	configData, err := loader.LoadConfig()
	if err != nil {
		return fmt.Errorf("loader.LoadConfig: %w", err)
	}

	// check that we have at least one argument and that it is help:
	if flag.NArg() == 1 && flag.Arg(0) == "help" {
		fmt.Println("Calvin - Google Calendar CLI, version", embeddedVersion)
		fmt.Println("Usage: calvin <username> <date>")
		fmt.Println("Example: calvin --local john.doe next wednesday")
		fmt.Println("         calvin john.doe [next] week")
		return nil
	}
	// if the there is one or more arguments, the first one is the username, if not, we fall back to the default username:
	var username string
	if flag.NArg() < 1 {
		if configData.DefaultUser == "" {
			return fmt.Errorf("no username specified and no default user in config")
		}
		username = configData.DefaultUser
	} else {
		username = flag.Arg(0)
	}
	// Parse username and date arguments
	parser := dateparse.New()
	parseResult, err := parser.Parse(flag.Args())
	if err != nil {
		return err
	}

	// Build the full calendar ID
	fullCalendarID := buildCalendarID(username, configData)

	// Initialize Google Calendar service
	gcalService, err := gcal.NewGCalService(loader)
	if err != nil {
		return fmt.Errorf("gcal.NewGCalService: %w", err)
	}

	// find time.location:
	var loc *time.Location

	if useLocalTimezone {
		loc = time.Local
		fmt.Println("Using local timezone:", loc)
	}

	// List and print events
	if parseResult.IsWeek {
		// If it's a week request, list events for the entire week
		if err := gcal.ListAndPrintEventsForWeek(gcalService, fullCalendarID, parseResult.WeekDays, configData.DefaultDomain, loc); err != nil {
			return fmt.Errorf("gcal.ListAndPrintEventsForWeek: %w", err)
		}
	} else {
		// Otherwise, list events for a single day
		if err := gcal.ListAndPrintEvents(gcalService, fullCalendarID, parseResult.Date, configData.DefaultDomain, loc); err != nil {
			return fmt.Errorf("gcal.ListAndPrintEvents: %w", err)
		}
	}

	return nil
}

// buildCalendarID constructs the calendar ID based on the username and default domain from config.
func buildCalendarID(username string, configData *config.Config) string {
	if containsAt(username) {
		return username
	}
	return fmt.Sprintf("%s@%s", username, configData.DefaultDomain)
}

// containsAt checks if the string contains '@'.
func containsAt(s string) bool {
	for _, c := range s {
		if c == '@' {
			return true
		}
	}
	return false
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}
