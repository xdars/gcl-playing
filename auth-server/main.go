package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"database/sql"
	gonanoid "github.com/matoous/go-nanoid/v2"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"io/ioutil"
	. "shared/jwt"
	"time"
)

type Client struct {
	ID     string `json:"client_id"`
	Secret string `json:"client_secret"`
}

var oauthConfig *oauth2.Config

func SetupOAuthConfig() {
	config, err := os.Open("cfg.json")
	if err != nil {
		fmt.Println(err)
	}
	defer config.Close()

	byteValue, _ := ioutil.ReadAll(config)

	var client Client

	json.Unmarshal(byteValue, &client)

	oauthConfig = &oauth2.Config{
		ClientID:     client.ID,
		ClientSecret: client.Secret,
		RedirectURL:  "http://localhost:3000/auth/callback",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/calendar",
		},
		Endpoint: google.Endpoint,
	}
}

func userExists(email string) bool {
	db, err := sql.Open("sqlite3", "../data.db")
	if err != nil {
		log.Println(err)
	}
	defer db.Close()

	stmt, err := db.Prepare("select token from users where email = ?")
	if err != nil {
		log.Println(err)
	}
	defer stmt.Close()
	var token string
	err = stmt.QueryRow(email).Scan(&token)
	if err != nil {
		fmt.Println("user does not exist")
		return false
	}
	if len(token) == 0 {
		log.Println("token is empty. = used does not exist")
	}

	return true
}

func addUser(email, token, rtoken string) {
	db, err := sql.Open("sqlite3", "../data.db")
	if err != nil {
		log.Println(err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Println(err)
	}
	stmt, err := tx.Prepare("insert into users(id, email, created_at, token, refresh_token, is_outlook) values(?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Println(err)
	}
	defer stmt.Close()
	id, err := gonanoid.New()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Generated id: %s\n", id)
	_, err = stmt.Exec(id, email, 0, token, rtoken, 0)
	if err != nil {
		log.Println(err)
	}
	err = tx.Commit()
	if err != nil {
		log.Println(err)
	}
}
func main() {

	SetupOAuthConfig()
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/auth", handleAuth)
	http.HandleFunc("/auth/callback", handleCallback)

	fmt.Println("Auth server started at :3000")
	http.ListenAndServe(":3000", nil)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	jwt, err := r.Cookie("JWT")
	if err == http.ErrNoCookie {
		http.Redirect(w, r, "/auth", http.StatusFound)
		return
	} else {
		if _, err := ParseJWT(jwt.Value); err == nil {
			http.Redirect(w, r, "http://localhost:8080/home", http.StatusFound)
			return
		}
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
		http.Error(w, "Token exchange failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Access Token: %s", token.AccessToken)
	log.Printf("Refresh Token: %s", token.RefreshToken)
	log.Printf("Expiry: %s", token.Expiry)
	client := oauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		http.Error(w, "Failed getting user info", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var userInfo struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		http.Error(w, "Failed decoding user info", http.StatusInternalServerError)
		return
	}

	if !userExists(userInfo.Email) {
		log.Printf("user %s does not exist. adding", userInfo.Email)
		addUser(userInfo.Email, token.AccessToken, token.RefreshToken)
	} else {
		log.Printf("user %s exists. doing nothing", userInfo.Email)
	}
	jwtToken, err := GenerateJWT(userInfo.Email)
	if err != nil {
		http.Error(w, "Failed to generate JWT", http.StatusInternalServerError)
		return
	}

	cookie := http.Cookie{
		Name:     "JWT",
		Value:    jwtToken,
		Expires:  time.Now().Add(3600 * time.Hour),
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
	}
	http.SetCookie(w, &cookie)
	redirectURL := fmt.Sprintf("http://localhost:5173/home")
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}
