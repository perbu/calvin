package gcal

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/perbu/calvin/config"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/fatih/color"
)

// GCalService implements CalendarService interface.
type GCalService struct {
	service *calendar.Service
	config  *config.Config
	loader  config.Loader
}

// NewGCalService initializes and returns a GCalService.
func NewGCalService(loader config.Loader) (*GCalService, error) {
	cfg, err := loader.LoadConfig()
	if err != nil {
		return nil, err
	}

	credBytes, err := loader.LoadCredentials()
	if err != nil {
		return nil, err
	}

	tokenBytes, err := loader.LoadToken()
	var tok *oauth2.Token
	if err != nil {
		tok, err = getTokenFromWeb(credBytes, loader)
		if err != nil {
			return nil, err
		}
	} else {
		if err := json.Unmarshal(tokenBytes, &tok); err != nil {
			return nil, fmt.Errorf("json.Unmarshal token: %w", err)
		}
	}

	conf, err := google.ConfigFromJSON(credBytes, calendar.CalendarReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %w", err)
	}

	client := conf.Client(context.Background(), tok)
	srv, err := calendar.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Calendar client: %w", err)
	}

	return &GCalService{
		service: srv,
		config:  cfg,
		loader:  loader,
	}, nil
}

// ListEvents retrieves events for a specific calendar and date.
func (g *GCalService) ListEvents(calendarID string, theDate time.Time) (*calendar.Events, error) {
	// Step 1: Retrieve calendar details to get the time zone.
	cal, err := g.service.Calendars.Get(calendarID).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve calendar info: %w", err)
	}

	// Step 2: Load the calendar's time zone.
	loc, err := time.LoadLocation(cal.TimeZone)
	if err != nil {
		return nil, fmt.Errorf("unable to load location from timezone %s: %w", cal.TimeZone, err)
	}

	// Step 3: Compute start and end of day based on the calendar's time zone.
	startOfDay := time.Date(theDate.Year(), theDate.Month(), theDate.Day(), 0, 0, 0, 0, loc)
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Fetch events using the computed times.
	events, err := g.service.Events.List(calendarID).
		ShowDeleted(false).
		SingleEvents(true).
		TimeMin(startOfDay.Format(time.RFC3339)).
		TimeMax(endOfDay.Format(time.RFC3339)).
		OrderBy("startTime").
		Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve events: %w", err)
	}
	return events, nil
}

// getTokenFromWeb handles OAuth2 authentication flow.
func getTokenFromWeb(credBytes []byte, loader config.Loader) (*oauth2.Token, error) {
	conf, err := google.ConfigFromJSON(credBytes, calendar.CalendarReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %w", err)
	}

	state := randomString(16)
	codeCh := make(chan string)
	srv := &http.Server{Addr: ":8066"}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			_, _ = fmt.Fprintln(w, "Invalid state")
			return
		}
		code := r.URL.Query().Get("code")
		_, _ = fmt.Fprintln(w, "Received authentication code. You can close this page now.")
		codeCh <- code
	})

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()

	authURL := conf.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("redirect_uri", "http://localhost:8066/"),
	)
	fmt.Printf("Go to the following link in your browser:\n%v\n", authURL)

	authCode := <-codeCh
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("HTTP server Shutdown: %v", err)
	}

	tok, err := conf.Exchange(context.Background(), authCode,
		oauth2.SetAuthURLParam("redirect_uri", "http://localhost:8066/"),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web: %w", err)
	}

	tokenBytes, err := json.Marshal(tok)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal token: %w", err)
	}
	if err := loader.SaveToken(tokenBytes); err != nil {
		return nil, fmt.Errorf("unable to save token: %w", err)
	}
	return tok, nil
}

// randomString generates a random string of the given length.
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))] // Simplistic for example
	}
	return string(b)
}

// ListAndPrintEvents handles the logic of listing and printing events.
func ListAndPrintEvents(s CalendarService, calendarID string, theDate time.Time, defaultDomain string) error {
	events, err := s.ListEvents(calendarID, theDate)

	if err != nil {
		return err
	}

	// Set up color helpers
	headerColor := color.New(color.FgCyan, color.Bold).SprintFunc()
	highlight := color.New(color.FgGreen).SprintFunc()
	subtle := color.New(color.FgHiBlack).SprintFunc()
	warnColor := color.New(color.FgRed, color.Bold).SprintFunc()

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
		var timeInfo string
		switch {
		case item.Start != nil && item.Start.Date != "":
			timeInfo = highlight("(all day)")
		case item.Start != nil && item.Start.DateTime != "":
			timeInfo = fmt.Sprintf("(%s --> %s)",
				highlight(extractTimeFromISO(item.Start.DateTime)),
				highlight(extractTimeFromISO(item.End.DateTime)),
			)
		}

		summaryColor := color.New(color.FgYellow, color.Bold).SprintFunc()

		fmt.Printf(" - %s %s %s\n",
			summaryColor(item.Summary),
			timeInfo,
			subtle("["+compactAttendees(item.Attendees, calendarID, defaultDomain)+"]"),
		)
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
