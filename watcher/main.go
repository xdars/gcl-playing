package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
)

type Config struct {
	BackendAddr   string
	BackendPort   string
	WatcherPort   string
	OutlookClid   string
	OutlookSecret string
}

var config Config
var oauthConf *oauth2.Config

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func main() {
	loadConfig()
	initOAuth()

	r := gin.Default()

	r.POST("/google/webhook", handleGoogleWebhook)

	log.Printf("Watcher running on port %s", config.WatcherPort)
	if err := r.Run(":" + config.WatcherPort); err != nil {
		log.Fatal("Failed to run server:", err)
	}
}

func loadConfig() {
	config = Config{
		BackendAddr:   getEnv("BACKEND_ADDR", "http://localhost"),
		BackendPort:   getEnv("BACKEND_PORT", "8080"),
		WatcherPort:   getEnv("WATCHER_PORT", "3030"),
		OutlookClid:   os.Getenv("OUTLOOK_CLIENT_ID"),
		OutlookSecret: os.Getenv("OUTLOOK_CLIENT_SECRET"),
	}
}

func initOAuth() {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		log.Fatal("GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET environment variables must be set")
	}

	redirectURI := getEnv("GOOGLE_REDIRECT_URI", "http://localhost:3030/auth/callback")

	oauthConf = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
		RedirectURL: redirectURI,
		Scopes:      []string{calendar.CalendarReadonlyScope},
	}
}

func handleGoogleWebhook(c *gin.Context) {
	resourceID := c.GetHeader("X-Goog-Resource-ID")
	if resourceID == "" {
		log.Println("Missing X-Goog-Resource-ID header")
		c.Status(http.StatusBadRequest)
		return
	}

	// TODO: Load calendar ID and tokens from database based on resourceID
	calendarID := "" // Load from DB
	at := ""         // Access token from DB
	rt := ""         // Refresh token from DB

	if calendarID == "" || at == "" {
		log.Println("Calendar not configured - webhook received but no calendar mapping")
		c.Status(http.StatusOK) // Acknowledge webhook but do nothing
		return
	}

	token := &oauth2.Token{
		AccessToken:  at,
		RefreshToken: rt,
		TokenType:    "Bearer",
	}

	ctx := context.Background()
	client := oauthConf.Client(ctx, token)

	srv, err := calendar.New(client)
	if err != nil {
		log.Printf("Calendar client error: %v", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	updatedMin := time.Now().Add(-30 * time.Second).Format(time.RFC3339)
	events, err := srv.Events.List(calendarID).
		ShowDeleted(false).
		SingleEvents(true).
		OrderBy("updated").
		TimeMin(updatedMin).
		Do()
	if err != nil {
		log.Printf("Failed to list events: %v", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	for _, event := range events.Items {
		log.Printf("Triggering /event/%s (%s)\n", event.Id, event.Summary)
		postToBackend("/event/"+event.Id, map[string]string{"summary": event.Summary})
	}

	c.Status(http.StatusOK)
}

func setWatcher(token, calendarID, webhookAddress string) error {
	payload := map[string]interface{}{
		"id":      "channel-" + calendarID,
		"type":    "web_hook",
		"address": webhookAddress,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST",
		"https://www.googleapis.com/calendar/v3/calendars/"+calendarID+"/events/watch",
		bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Watcher setup failed: %s - %s", resp.Status, string(body))
	} else {
		log.Printf("Watcher set for calendar: %s", calendarID)
	}

	return nil
}

func postToBackend(endpoint string, payload any) {
	url := config.BackendAddr + ":" + config.BackendPort + endpoint

	resp, err := http.Post(url, "application/json", toReader(payload))
	if err != nil {
		log.Printf("POST to backend failed: %v", err)
		return
	}
	defer resp.Body.Close()
}

func toReader(v any) io.Reader {
	buf, _ := json.Marshal(v)
	return bytes.NewReader(buf)
}

func splitResource(s string) []string {
	return strings.Split(s, "/")
}
