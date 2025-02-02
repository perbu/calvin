package main

import (
	_ "embed"
	"fmt"
	"github.com/perbu/calvin/config"
	"github.com/perbu/calvin/dateparse"
	"github.com/perbu/calvin/gcal"
	"log"
	"os"
)

//go:embed .version
var embeddedVersion string

func run(args []string) error {
	// Initialize configuration loader
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
	if len(args) < 1 || args[0] == "help" {
		fmt.Println("Calvin - Google Calendar CLI, version", embeddedVersion)
		fmt.Println("Usage: calvin <username> <date>")
		fmt.Println("Example: calvin john.doe next wednesday")
		return nil
	}

	// Parse command-line arguments
	parser := dateparse.New()
	username, theDate, err := parser.Parse(args)
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

	// List and print events
	if err := gcal.ListAndPrintEvents(gcalService, fullCalendarID, theDate, configData.DefaultDomain); err != nil {
		return fmt.Errorf("gcal.ListAndPrintEvents: %w", err)
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
