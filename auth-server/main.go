package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"shared/database"
	. "shared/jwt"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found, using environment variables")
	}
}

var oauthConfig *oauth2.Config

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func SetupOAuthConfig() {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		log.Fatal("GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET environment variables must be set")
	}

	redirectURL := getEnv("OAUTH_REDIRECT_URL", "http://localhost:3000/auth/callback")

	oauthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/calendar",
		},
		Endpoint: google.Endpoint,
	}
}

func userExists(email string) bool {
	db, err := database.GetDB()
	if err != nil {
		log.Printf("Failed to get database: %v", err)
		return false
	}

	var token string
	err = db.QueryRow("SELECT token FROM users WHERE email = ?", email).Scan(&token)
	if err != nil {
		return false
	}
	return true
}

func addUser(email, token, rtoken string) error {
	db, err := database.GetDB()
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	id, err := gonanoid.New()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to generate id: %w", err)
	}

	_, err = tx.Exec(
		"INSERT INTO users(id, email, created_at, token, refresh_token, is_outlook) VALUES(?, ?, ?, ?, ?, ?)",
		id, email, time.Now().Unix(), token, rtoken, 0,
	)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to insert user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	log.Printf("Added user: %s", email)
	return nil
}

func updateUserTokens(email, token, rtoken string) error {
	db, err := database.GetDB()
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}

	_, err = db.Exec(
		"UPDATE users SET token = ?, refresh_token = ? WHERE email = ?",
		token, rtoken, email,
	)
	if err != nil {
		return fmt.Errorf("failed to update user tokens: %w", err)
	}

	log.Printf("Updated tokens for user: %s", email)
	return nil
}

func main() {
	SetupOAuthConfig()

	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/auth", handleAuth)
	http.HandleFunc("/auth/callback", handleCallback)

	port := getEnv("AUTH_SERVER_PORT", "3000")
	fmt.Printf("Auth server started at :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	jwt, err := r.Cookie("JWT")
	if err == http.ErrNoCookie {
		http.Redirect(w, r, "/auth", http.StatusFound)
		return
	}

	if _, err := ParseJWT(jwt.Value); err == nil {
		backendURL := getEnv("BACKEND_URL", "http://localhost:8080")
		http.Redirect(w, r, backendURL+"/home", http.StatusFound)
		return
	}

	http.Redirect(w, r, "/auth", http.StatusFound)
}

func handleAuth(w http.ResponseWriter, r *http.Request) {
	url := oauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	http.Redirect(w, r, url, http.StatusFound)
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing code", http.StatusBadRequest)
		return
	}

	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		log.Printf("Token exchange failed: %v", err)
		http.Error(w, "Token exchange failed", http.StatusInternalServerError)
		return
	}

	client := oauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		log.Printf("Failed getting user info: %v", err)
		http.Error(w, "Failed getting user info", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var userInfo struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		log.Printf("Failed decoding user info: %v", err)
		http.Error(w, "Failed decoding user info", http.StatusInternalServerError)
		return
	}

	if !userExists(userInfo.Email) {
		log.Printf("New user: %s", userInfo.Email)
		if err := addUser(userInfo.Email, token.AccessToken, token.RefreshToken); err != nil {
			log.Printf("Failed to add user: %v", err)
			http.Error(w, "Failed to save user", http.StatusInternalServerError)
			return
		}
	} else {
		log.Printf("Existing user: %s", userInfo.Email)
		if err := updateUserTokens(userInfo.Email, token.AccessToken, token.RefreshToken); err != nil {
			log.Printf("Failed to update user tokens: %v", err)
		}
	}

	jwtToken, err := GenerateJWT(userInfo.Email)
	if err != nil {
		log.Printf("Failed to generate JWT: %v", err)
		http.Error(w, "Failed to generate JWT", http.StatusInternalServerError)
		return
	}

	backendURL := getEnv("BACKEND_URL", "http://localhost:8080")
	http.Redirect(w, r, backendURL+"/auth/callback?token="+jwtToken, http.StatusSeeOther)
}
