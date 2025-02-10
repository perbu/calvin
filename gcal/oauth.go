package gcal

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/perbu/calvin/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"log"
	"net/http"
	"time"
)

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
