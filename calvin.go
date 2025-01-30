package main

import (
	"context"
	_ "embed"
	"encoding/json"
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
)

//go:embed credentials.json
var googleCredentials []byte

type Config struct {
	DefaultDomain string `json:"default_domain"`
}

// getConfigPath returns the path to our CLI config directory (e.g., ~/.mycal).
func getConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Unable to find user home directory: %v", err)
	}
	return filepath.Join(homeDir, ".calvin")
}

// getConfig loads the configuration from ~/.mycal/config.json.
func getConfig() (Config, error) {
	var config Config

	configDir := getConfigPath()
	configPath := filepath.Join(configDir, "config.json")
	b, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, fmt.Errorf("os.ReadFile(%s): %w", configPath, err)
	}
	err = json.Unmarshal(b, &config)
	if err != nil {
		return Config{}, fmt.Errorf("json.Unmarshal: %w", err)
	}
	return config, nil
}

// getClient handles OAuth2 flow, returning an authorized calendar.Service.
func getClient() *calendar.Service {
	// The token will be saved/read at ~/.mycal/token.json
	configDir := getConfigPath()
	if err := os.MkdirAll(configDir, 0700); err != nil {
		log.Fatalf("Unable to create config directory: %v", err)
	}
	tokenPath := filepath.Join(configDir, "token.json")

	// If modifying scopes, delete your previously saved token.json.
	conf, err := google.ConfigFromJSON(googleCredentials, calendar.CalendarReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	tok, err := tokenFromFile(tokenPath)
	if err != nil {
		// If there's no valid token file, we do the web-based auth flow
		tok = getTokenFromWeb(conf, tokenPath)
	}
	client := conf.Client(context.Background(), tok)
	srv, err := calendar.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Calendar client: %v", err)
	}
	return srv
}

// tokenFromFile tries to read the OAuth2 token from local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// saveToken saves OAuth2 token to a local file.
func saveToken(path string, token *oauth2.Token) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("os.Create: %w", err)
	}
	defer f.Close()
	err = json.NewEncoder(f).Encode(token)
	if err != nil {
		return fmt.Errorf("json.NewEncoder.Encode: %w", err)
	}
	return nil
}

// getTokenFromWeb runs a small local webserver to get the OAuth2 code from Google.
func getTokenFromWeb(conf *oauth2.Config, tokenPath string) *oauth2.Token {
	state := randomString(16)
	codeCh := make(chan string)
	srv := &http.Server{Addr: ":8066"}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			_, _ = fmt.Fprintf(w, "Invalid state")
			return
		}
		code := r.URL.Query().Get("code")
		_, _ = fmt.Fprintln(w, "Received authentication code. You can close this page now.")
		codeCh <- code
	})

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
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
		oauth2.SetAuthURLParam("redirect_uri", "http://localhost:8066/"))
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	if err := saveToken(tokenPath, tok); err != nil {
		log.Fatalf("Unable to save token: %v", err)
	}
	return tok
}

// randomString returns a random string of the specified length (for state).
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// parseArgs parses command-line arguments to get the username and optional date.
//
// Usage examples:
//
//	mycal bob
//	mycal bob --date=2025-03-15
//	mycal bob --date=tomorrow
func parseArgs() (string, time.Time, error) {
	dateFlag := flag.String("date", "", "Date in format YYYY-MM-DD or 'tomorrow'")
	flag.Parse()

	// We expect exactly one non-flag argument: the username
	args := flag.Args()
	if len(args) != 1 {
		return "", time.Time{}, fmt.Errorf("usage: %s <username> [--date=YYYY-MM-DD|tomorrow]", os.Args[0])
	}
	username := args[0]

	// If no date is provided, default to today
	theDate := time.Now().Truncate(24 * time.Hour)

	// If --date=tomorrow is passed
	if *dateFlag == "tomorrow" {
		theDate = theDate.Add(24 * time.Hour)
	} else if *dateFlag != "" {
		// If an ISO string is passed, parse it
		parsed, err := time.Parse("2006-01-02", *dateFlag)
		if err == nil {
			theDate = parsed
		} else {
			// If we can't parse the provided date, fallback to today's date
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

// listEvents queries the Calendar API for all events on `theDate`.
func listEvents(service *calendar.Service, calID string, theDate time.Time) error {
	// Start and end of day in local time
	startOfDay := time.Date(theDate.Year(), theDate.Month(), theDate.Day(), 0, 0, 0, 0, time.Local)
	endOfDay := startOfDay.Add(24 * time.Hour)

	fmt.Printf("Listing events for %s (%s) ...\n",
		theDate.Format("2006-01-02"),
		calID,
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
		fmt.Println("No events found.")
		return nil
	}
	// Print each event
	for _, item := range events.Items {
		var timeInfo string
		switch {
		// If it's an all-day event, `Date` field is set.
		case item.Start != nil && item.Start.Date != "":
			timeInfo = "(all day)"
		// Otherwise, we can show the times in local or TZ
		case item.Start != nil && item.Start.DateTime != "":
			timeInfo = fmt.Sprintf("(%s --> %s)",
				item.Start.DateTime,
				item.End.DateTime,
			)
		}
		fmt.Printf(" - %s %s\n", item.Summary, timeInfo)
	}
	return nil
}

func main() {
	username, theDate, err := parseArgs()
	if err != nil {
		log.Fatalln(err)
	}

	// Load configuration to get a default domain
	config, err := getConfig()
	if err != nil {
		log.Fatalf("unable to load config file: %v", err)
	}

	// Build the full calendar ID (email address)
	calendarID := buildCalendarID(username, config.DefaultDomain)

	// Get an authenticated client
	svc := getClient()

	// List the events
	err = listEvents(svc, calendarID, theDate)
	if err != nil {
		log.Fatalln("Error listing events:", err)
	}
}
