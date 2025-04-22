package gcal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/fatih/color"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/perbu/calvin/config"
)

const (
	separatorCount = 8
)

// GCalService interacts with the Google Calendar API.
type GCalService struct {
	service *calendar.Service
	config  *config.Config
	loader  config.Loader
}

// NewGCalService creates and initializes a new GCalService.
func NewGCalService(loader config.Loader) (*GCalService, error) {
	cfg, err := loader.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	credBytes, err := loader.LoadCredentials()
	if err != nil {
		return nil, fmt.Errorf("loading credentials: %w", err)
	}

	token, err := loadOrObtainToken(credBytes, loader)
	if err != nil {
		return nil, fmt.Errorf("getting token: %w", err)
	}

	client := oauthClient(credBytes, token)

	srv, err := calendar.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("creating calendar service: %w", err)
	}

	return &GCalService{service: srv, config: cfg, loader: loader}, nil
}

// loadOrObtainToken loads a token from storage or obtains a new one if necessary.
func loadOrObtainToken(credBytes []byte, loader config.Loader) (*oauth2.Token, error) {
	tokenBytes, err := loader.LoadToken()
	if err == nil { // Token found in storage
		var tok oauth2.Token
		if err := json.Unmarshal(tokenBytes, &tok); err != nil {
			return nil, fmt.Errorf("unmarshalling token: %w", err)
		}
		return &tok, nil
	}

	// No token found, initiate OAuth2 flow
	return getTokenFromWeb(credBytes, loader)
}

// oauthClient creates an OAuth2 client.
func oauthClient(credBytes []byte, token *oauth2.Token) *http.Client {
	conf, err := google.ConfigFromJSON(credBytes, calendar.CalendarReadonlyScope)
	if err != nil {
		log.Fatalf("parsing credentials: %v", err) // Fatal error if credentials are invalid
	}
	return conf.Client(context.Background(), token)
}

// ListEvents retrieves events for a given calendar ID and date.
func (g *GCalService) ListEvents(calendarID string, theDate time.Time) (*calendar.Events, error) {
	cal, err := g.service.Calendars.Get(calendarID).Do()
	if err != nil {
		return nil, fmt.Errorf("getting calendar info: %w", err)
	}

	loc, err := time.LoadLocation(cal.TimeZone)
	if err != nil {
		return nil, fmt.Errorf("loading location: %w", err)
	}

	startOfDay := time.Date(theDate.Year(), theDate.Month(), theDate.Day(), 0, 0, 0, 0, loc)
	endOfDay := startOfDay.Add(24 * time.Hour)

	events, err := g.service.Events.List(calendarID).
		ShowDeleted(false).
		SingleEvents(true).
		TimeMin(startOfDay.Format(time.RFC3339)).
		TimeMax(endOfDay.Format(time.RFC3339)).
		OrderBy("startTime").
		Do()
	if err != nil {
		return nil, fmt.Errorf("retrieving events: %w", err)
	}
	return events, nil
}

// formatTimeInfo formats the time information for an event.
func formatTimeInfo(item *calendar.Event, loc *time.Location) string {
	if item.Start == nil {
		return "" // Handle cases where Start is nil for robustness
	}

	if item.Start.Date != "" {
		return color.New(color.FgGreen).SprintFunc()("(all day)")
	}

	if item.Start.DateTime != "" {
		startTime, err1 := time.Parse(time.RFC3339, item.Start.DateTime)
		endTime, err2 := time.Parse(time.RFC3339, item.End.DateTime)

		if err1 != nil || err2 != nil {
			return fmt.Sprintf("(%s --> %s)",
				extractTimeFromISO(item.Start.DateTime),
				extractTimeFromISO(item.End.DateTime),
			) // Fallback if parsing fails
		}

		highlight := color.New(color.FgGreen).SprintFunc()
		var formatted string
		if loc == nil {
			formatted = fmt.Sprintf(" [%s --> %s]", highlight(startTime.Format("15:04")), highlight(endTime.Format("15:04")))
		} else {
			formatted = fmt.Sprintf(" [%s --> %s]", highlight(startTime.In(loc).Format("15:04")), highlight(endTime.In(loc).Format("15:04")))
		}

		return formatted
	}

	return "" // Default return if no time information is available
}

// ListAndPrintEvents lists and prints events for a given calendar and date.
func ListAndPrintEvents(s CalendarService, calendarID string, theDate time.Time, defaultDomain string, loc *time.Location) error {
	events, err := s.ListEvents(calendarID, theDate)
	if err != nil {
		return err
	}

	headerColor := color.New(color.FgCyan, color.Bold).SprintFunc()
	warnColor := color.New(color.FgRed, color.Bold).SprintFunc()
	subtle := color.New(color.FgHiBlack).SprintFunc()
	summaryColor := color.New(color.FgYellow, color.Bold).SprintFunc()

	fmt.Printf("Listing events for %s (%s) [tz: %s]...\n",
		headerColor(theDate.Format("2006-01-02")),
		headerColor(calendarID),
		headerColor(events.TimeZone),
	)

	if len(events.Items) == 0 {
		fmt.Println(warnColor("No events found."))
		return nil
	}

	for _, item := range events.Items {
		fmt.Printf(" - %s %s %s %s\n",
			summaryColor(item.Summary),
			formatTimeInfo(item, loc), // Call the helper function
			subtle("["+compactAttendees(item.Attendees, calendarID, defaultDomain)+"]"),
			extractURLs(item), // Call the helper function
		)
	}
	return nil
}

// ListAndPrintEventsForWeekDay lists and prints events for a given calendar and date with a simplified header for week view.
func ListAndPrintEventsForWeekDay(s CalendarService, calendarID string, theDate time.Time, defaultDomain string, loc *time.Location) error {
	events, err := s.ListEvents(calendarID, theDate)
	if err != nil {
		return err
	}

	headerColor := color.New(color.FgCyan, color.Bold).SprintFunc()
	warnColor := color.New(color.FgRed, color.Bold).SprintFunc()
	subtle := color.New(color.FgHiBlack).SprintFunc()
	summaryColor := color.New(color.FgYellow, color.Bold).SprintFunc()

	// Simplified header for week view - only show the date
	fmt.Printf("%s:\n", headerColor(theDate.Format("=== Monday (Jan 2) ===")))

	if len(events.Items) == 0 {
		fmt.Println(warnColor("No events found."))
		return nil
	}

	for _, item := range events.Items {
		fmt.Printf(" - %s %s %s %s\n",
			summaryColor(item.Summary),
			formatTimeInfo(item, loc),
			subtle("["+compactAttendees(item.Attendees, calendarID, defaultDomain)+"]"),
			extractURLs(item),
		)
	}
	return nil
}

// ListAndPrintEventsForWeek lists and prints events for a given calendar for each day in a week.
func ListAndPrintEventsForWeek(s CalendarService, calendarID string, weekDays []time.Time, defaultDomain string, loc *time.Location) error {
	// Get the first day's events to extract timezone information
	firstDayEvents, err := s.ListEvents(calendarID, weekDays[0])
	if err != nil {
		return err
	}

	headerColor := color.New(color.FgCyan, color.Bold).SprintFunc()

	fmt.Printf("Listing events for the week of %s to %s (%s) [tz: %s]\n",
		headerColor(weekDays[0].Format("2006-01-02")),
		headerColor(weekDays[6].Format("2006-01-02")),
		headerColor(calendarID),
		headerColor(firstDayEvents.TimeZone))

	// fmt.Println(strings.Repeat("-", separatorCount))

	for _, day := range weekDays {
		err := ListAndPrintEventsForWeekDay(s, calendarID, day, defaultDomain, loc)
		if err != nil {
			return err
		}
		// fmt.Println(strings.Repeat("-", separatorCount))
	}

	return nil
}

// extractTimeFromISO converts ISO time to "15:04" format.
func extractTimeFromISO(isoDateTime string) string {
	t, err := time.Parse(time.RFC3339, isoDateTime)
	if err != nil {
		return "[error parsing time]"
	}
	return t.Format("15:04")
}

func compactAttendees(attendees []*calendar.EventAttendee, self, homeDomain string) string {
	if len(attendees) == 0 {
		return ""
	}
	var who []string
	for _, a := range attendees {
		if a.Email == self {
			continue
		}
		short := strings.TrimSuffix(a.Email, "@"+homeDomain)
		who = append(who, short)
		if len(who) >= 3 {
			who = append(who, "...")
			break
		}
	}
	return strings.Join(who, ", ")
}

func extractURLs(item *calendar.Event) string {
	if item.HangoutLink != "" {
		return item.HangoutLink
	}
	if item.Location != "" {
		return item.Location
	}
	return ""
}
