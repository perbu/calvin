package main

import (
	"fmt"
	"github.com/perbu/calvin/config"
	"github.com/perbu/calvin/dateparse"
	"github.com/perbu/calvin/gcal"
	"log"
	"os"
)

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
