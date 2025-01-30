package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	// Added color package
	"github.com/fatih/color"
)

const (
	tokenFileName = "token.json"
)

type Config struct {
	DefaultDomain string `json:"default_domain"`
}

// getConfigPath returns the path to our CLI config directory (e.g., ~/.calvin).
func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to find user home directory: %w", err)
	}
	return filepath.Join(homeDir, ".calvin"), nil
}

// getConfig loads the configuration from ~/.calvin/config.json.
func getConfig() (Config, error) {
	configDir, err := getConfigPath()
	if err != nil {
		return Config{}, err
	}
	configPath := filepath.Join(configDir, "config.json")

	b, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, fmt.Errorf("os.ReadFile(%s): %w", configPath, err)
	}

	var config Config
	if err := json.Unmarshal(b, &config); err != nil {
		return Config{}, fmt.Errorf("json.Unmarshal: %w", err)
	}
	return config, nil
}

func getCredentials() ([]byte, error) {
	configDir, err := getConfigPath()
	if err != nil {
		return nil, err
	}
	credentialsPath := filepath.Join(configDir, "credentials.json")
	bytes, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("os.ReadFile(%s): %w", credentialsPath, err)
	}
	return bytes, nil
}

// newCalendarService handles OAuth2 flow, returning an authorized calendar.Service.
func newCalendarService() (*calendar.Service, error) {
	configDir, err := getConfigPath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return nil, fmt.Errorf("unable to create config directory: %w", err)
	}
	tokenPath := filepath.Join(configDir, tokenFileName)
	credBytes, err := getCredentials()
	if err != nil {
		return nil, err
	}
	// If modifying scopes, delete your previously saved token.json.
	conf, err := google.ConfigFromJSON(credBytes, calendar.CalendarReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %w", err)
	}

	tok, err := tokenFromFile(tokenPath)
	if err != nil {
		// If there's no valid token file, do the web-based auth flow
		tok, err = getTokenFromWeb(conf, tokenPath)
		if err != nil {
			return nil, err
		}
	}

	client := conf.Client(context.Background(), tok)
	srv, err := calendar.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Calendar client: %w", err)
	}
	return srv, nil
}

// tokenFromFile tries to read the OAuth2 token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tok := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(tok); err != nil {
		return nil, err
	}
	return tok, nil
}

// saveToken saves OAuth2 token to a local file.
func saveToken(path string, token *oauth2.Token) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("os.Create: %w", err)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(token); err != nil {
		return fmt.Errorf("json.NewEncoder.Encode: %w", err)
	}
	return nil
}

// getTokenFromWeb runs a small local webserver to get the OAuth2 code from Google.
func getTokenFromWeb(conf *oauth2.Config, tokenPath string) (*oauth2.Token, error) {
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
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(fmt.Sprintf("ListenAndServe error: %v", err))
		}
	}()

	// redirect_uri must match the URI in your Google Cloud Console
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

	tok, err := conf.Exchange(context.TODO(), authCode,
		oauth2.SetAuthURLParam("redirect_uri", "http://localhost:8066/"),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web: %w", err)
	}
	if err := saveToken(tokenPath, tok); err != nil {
		return nil, fmt.Errorf("unable to save token: %w", err)
	}
	return tok, nil
}

// randomString returns a random string of the specified length (for OAuth2 state).
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func parseArgs(args []string) (string, time.Time, error) {
	fs := flag.NewFlagSet("calvin", flag.ExitOnError)
	dateFlag := fs.String("date", "", "Date in format YYYY-MM-DD or 'tomorrow'")
	if err := fs.Parse(args); err != nil {
		return "", time.Time{}, err
	}

	// We expect exactly one non-flag argument: the username
	rem := fs.Args()
	if len(rem) != 1 {
		return "", time.Time{},
			fmt.Errorf("usage: %s <username> [--date=YYYY-MM-DD|tomorrow]", fs.Name())
	}
	username := rem[0]

	// If no date is provided, default to today
	theDate := time.Now().Truncate(24 * time.Hour)

	// If --date=tomorrow is passed
	switch *dateFlag {
	case "":
		// keep today's date
	case "tomorrow":
		theDate = theDate.Add(24 * time.Hour)
	default:
		parsed, err := time.Parse("2006-01-02", *dateFlag)
		if err == nil {
			theDate = parsed
		} else {
			log.Printf("Warning: could not parse date %q, using today", *dateFlag)
		}
	}
	return username, theDate, nil
}

// buildCalendarID checks if username contains '@'. If not, it appends the default domain.
func buildCalendarID(username, defaultDomain string) string {
	if strings.ContainsRune(username, '@') {
		return username
	}
	return fmt.Sprintf("%s@%s", username, defaultDomain)
}

func extractTimeFromISO(isoDateTime string) string {
	t, err := time.Parse(time.RFC3339, isoDateTime)
	if err != nil {
		return "[error parsing time]"
	}
	return t.Format("15:04")
}

// listEvents queries the Calendar API for all events on `theDate`.
func listEvents(service *calendar.Service, calID string, theDate time.Time, homeDomain string) error {
	// Start and end of day in local time
	startOfDay := time.Date(theDate.Year(), theDate.Month(), theDate.Day(), 0, 0, 0, 0, time.Local)
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Set up some color helpers
	headerColor := color.New(color.FgCyan, color.Bold).SprintFunc()
	highlight := color.New(color.FgGreen).SprintFunc()
	subtle := color.New(color.FgHiBlack).SprintFunc()
	warnColor := color.New(color.FgRed, color.Bold).SprintFunc()

	fmt.Printf("Listing events for %s (%s) ...\n",
		headerColor(theDate.Format("2006-01-02")),
		headerColor(calID),
	)

	events, err := service.Events.List(calID).
		ShowDeleted(false).
		SingleEvents(true).
		TimeMin(startOfDay.Format(time.RFC3339)).
		TimeMax(endOfDay.Format(time.RFC3339)).
		OrderBy("startTime").
		Do()
	if err != nil {
		return fmt.Errorf("unable to retrieve events: %w", err)
	}

	if len(events.Items) == 0 {
		// Print 'no events' in a red/bold style for emphasis
		fmt.Println(warnColor("No events found."))
		return nil
	}

	// Print each event in a color-coded line
	for _, item := range events.Items {
		var timeInfo string
		switch {
		case item.Start != nil && item.Start.Date != "":
			// all-day event
			timeInfo = highlight("(all day)")
		case item.Start != nil && item.Start.DateTime != "":
			timeInfo = fmt.Sprintf("(%s --> %s)",
				highlight(extractTimeFromISO(item.Start.DateTime)),
				highlight(extractTimeFromISO(item.End.DateTime)),
			)
		}

		// item.Summary in bold or another color
		summaryColor := color.New(color.FgYellow, color.Bold).SprintFunc()

		fmt.Printf(" - %s %s %s\n",
			summaryColor(item.Summary),
			timeInfo,
			subtle("["+compactAttendees(item.Attendees, calID, homeDomain)+"]"),
		)
	}
	return nil
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

// run is the main application logic, but returns errors instead of exiting.
func run(args []string) error {
	username, theDate, err := parseArgs(args)
	if err != nil {
		return err
	}

	// Load configuration to get a default domain
	config, err := getConfig()
	if err != nil {
		return fmt.Errorf("unable to load config file: %w", err)
	}

	// Build the full calendar ID (email address)
	calendarID := buildCalendarID(username, config.DefaultDomain)

	// Get an authenticated calendar client
	svc, err := newCalendarService()
	if err != nil {
		return err
	}

	// List the events
	if err := listEvents(svc, calendarID, theDate, config.DefaultDomain); err != nil {
		return fmt.Errorf("error listing events: %w", err)
	}

	return nil
}

// main is just a thin wrapper around run().
func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}
