package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"fmt"
	"io/ioutil"
	"bytes"
)

type Config struct {
	BackendAddr        string `json:"backendAddr"`
	BackendPort        string `json:"backendPort"`
	WatcherPort        string `json:"watcherPort"`
	OutlookClid        string `json:"outlook_clid"`
	OutlookSecret      string `json:"outlook_secret"`
	GoogleClientID     string `json:"google_client_id"`
	GoogleClientSecret string `json:"google_client_secret"`
	GoogleRedirectURI  string `json:"google_redirect_uri"`
}

var config Config
var oauthConf *oauth2.Config

/*
	curl -X POST \
  -H "Authorization: Bearer <ACCESS_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "your-channel-id-uuid",
    "type": "web_hook",
    "address": ""
  }' \
  "https://www.googleapis.com/calendar/v3/calendars/<calendarId>/events/watch"
*/

func setWatcher(token, calendarID string) {
	/*_, err := http.Post("https://www.googleapis.com/calendar/v3/calendars/"+calendarID+"/events/watch", "application/json",
		toReader(gin.H{
			"id":      "someuniquestuff",
			"type":    "web_hook",
			"address": "",
		},
		),
	)
	if err != nil {
		log.Printf("Setting watcher failed: %v", err)
		return
	}*/

	payload := map[string]interface{}{
		"id":      "someuniquestuff",
		"type":    "web_hook",
		"address": "",
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("Error marshaling JSON: %v", err)
	}

	req, err := http.NewRequest("POST", "https://www.googleapis.com/calendar/v3/calendars/"+calendarID+"/events/watch", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v", err)
	}

	fmt.Printf("Response Status: %s\n", resp.Status)
	fmt.Printf("Response Body: %s\n", body)
}
func main() {
	loadConfig()
	initOAuth()

	r := gin.Default()
	// This is supposed to set a watcher on each user we have in our DB, it not set yet.
	setWatcher(at, calendarID)

	/*r.POST("/event/:eventId", handleEvent)
	r.POST("/changewh/:at/:rt/:email", handleChangeWebhook)
	r.POST("/mergewh/:at/:rt/:email", handleMergeWebhook)
	r.POST("/outlook/webhook/:email", handleOutlookWebhook)*/
	r.POST("/google/webhook", handleGoogleWebhook)

	log.Printf("Watcher running on port %s", config.WatcherPort)
	if err := r.Run(":" + config.WatcherPort); err != nil {
		log.Fatal("Failed to run server:", err)
	}
}

func initOAuth() {
	oauthConf = &oauth2.Config{
		ClientID:     config.GoogleClientID,
		ClientSecret: config.GoogleClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
		RedirectURL: config.GoogleRedirectURI,
		Scopes:      []string{calendar.CalendarReadonlyScope},
	}
}

func handleGoogleWebhook(c *gin.Context) {
	resourceID := c.GetHeader("X-Goog-Resource-ID")
	if resourceID == "" {
		c.Status(http.StatusBadRequest)
		return
	}
	calendarID := "@gmail.com"                                                                                                                                                                                         // допустим
	at := "access_token" // load from DB by calendarID                                                                                                                          // refresh_token
	token := &oauth2.Token{
		AccessToken:  at,
		RefreshToken: rt,
		TokenType:    "Bearer",
	}

	ctx := context.Background()
	client := oauthConf.Client(ctx, token)

	srv, err := calendar.New(client)
	if err != nil {
		log.Println("calendar client error:", err)
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
		log.Println("failed to list events:", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	for _, event := range events.Items {
		log.Printf("Triggering /event/%s (%s)\n", event.Id, event.Summary)
		postToBackend("/event/"+event.Id, gin.H{"summary": event.Summary})
	}

	c.Status(http.StatusOK)
}

func loadConfig() {
	data, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatal("Failed to read config.json:", err)
	}
	if err := json.Unmarshal(data, &config); err != nil {
		log.Fatal("Invalid config.json:", err)
	}
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
	return io.NopCloser(io.Reader(&bufReader{data: buf}))
}

type bufReader struct {
	data []byte
	pos  int
}

func (r *bufReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func splitResource(s string) []string {
	return strings.Split(s, "/")
}

func decodeBase64Param(p string) string {
	decoded, err := base64.StdEncoding.DecodeString(p)
	if err != nil {
		return ""
	}
	return string(decoded)
}
