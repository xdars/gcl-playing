package handler

import (
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"

	"calendar-backend/config"
	"calendar-backend/database"
	"calendar-backend/google"
	"shared/jwt"
	"shared/logger"
)

var calendarService *google.CalendarService

func init() {
}

func InitCalendarService() {
	cfg := config.Cfg
	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
		calendarService = google.NewCalendarService(cfg.GoogleClientID, cfg.GoogleClientSecret)
		logger.Info.Printf("Google Calendar service initialized")
	} else {
		logger.Warn.Printf("Google Calendar service NOT initialized - missing GOOGLE_CLIENT_ID or GOOGLE_CLIENT_SECRET")
	}
}

func getAuthenticatedUser(c *gin.Context) *database.User {
	jwtCookie, err := c.Cookie("JWT")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return nil
	}

	email, err := jwt.ParseJWT(jwtCookie)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return nil
	}

	user, err := database.GetUser(email)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return nil
	}

	return user
}

func HandleEvent(c *gin.Context) {
	eventId := c.Param("eventId")
	logger.Info.Printf("Received event: %s", eventId)

	c.JSON(http.StatusOK, gin.H{
		"message": "Event received",
		"eventId": eventId,
	})
}

func HandleTokens(c *gin.Context) {
	user := getAuthenticatedUser(c)
	if user == nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  user.Token,
		"refresh_token": user.RefreshToken,
	})
}

func HandleAuthCallback(c *gin.Context) {
	newToken := c.Query("token")
	if newToken == "" {
		logger.Warn.Printf("Auth callback missing token")
		c.String(http.StatusBadRequest, "Missing token")
		return
	}

	newEmail, err := jwt.ParseJWT(newToken)
	if err != nil {
		logger.Warn.Printf("Invalid token in callback: %v", err)
		c.String(http.StatusBadRequest, "Invalid token")
		return
	}

	cfg := config.Cfg

	existingJWT, _ := c.Cookie("JWT")
	var existingUser *database.User

	if existingJWT != "" {
		existingEmail, err := jwt.ParseJWT(existingJWT)
		if err == nil {
			existingUser, _ = database.GetUser(existingEmail)
		}
	}

	if existingUser != nil && existingUser.Email != newEmail {
		// CONNECT FLOW: User is adding another Google account
		newUser, err := database.GetUser(newEmail)
		if err != nil || newUser == nil {
			logger.Error.Printf("New user not found: %s", newEmail)
			c.Redirect(http.StatusSeeOther, cfg.FrontendURL+"/home?error=account_not_found")
			return
		}

		// Check if this account is already connected
		existing, err := database.GetConnectedAccountByProviderAccountId(
			existingUser.ID, "google", newUser.ID,
		)

		if err != nil {
			logger.Error.Printf("Failed to check connected account: %v", err)
			c.Redirect(http.StatusSeeOther, cfg.FrontendURL+"/home?error="+url.QueryEscape("Failed to connect account"))
			return
		}

		if existing != nil {
			// Update existing connected account tokens
			err = database.UpdateConnectedAccountTokens(
				existing.ID,
				newUser.Token,
				newUser.RefreshToken,
				nil,
			)
			if err != nil {
				logger.Error.Printf("Failed to update connected account tokens: %v", err)
			}
		} else {
			// Create new connected account
			_, err = database.CreateConnectedAccount(database.ConnectedAccount{
				UserID:            existingUser.ID,
				Provider:          "google",
				ProviderAccountID: newUser.ID,
				Email:             newEmail,
				AccessToken:       newUser.Token,
				RefreshToken:      newUser.RefreshToken,
			})
			if err != nil {
				logger.Error.Printf("Failed to create connected account: %v", err)
				c.Redirect(http.StatusSeeOther, cfg.FrontendURL+"/home?error="+url.QueryEscape("Failed to connect account"))
				return
			}
		}

		// Keep existing JWT, redirect with success
		c.Redirect(http.StatusSeeOther, cfg.FrontendURL+"/home?connected="+url.QueryEscape(newEmail))
		return
	}

	// LOGIN FLOW: Set new JWT cookie
	c.SetCookie(
		"JWT",
		newToken,
		int(24*time.Hour.Seconds()),
		"/",
		"",
		false,
		true,
	)

	c.Redirect(http.StatusSeeOther, cfg.FrontendURL+"/home")
}


func HandleGetConnectedAccounts(c *gin.Context) {
	user := getAuthenticatedUser(c)
	if user == nil {
		return
	}

	accounts, err := database.GetConnectedAccountsByUserId(user.ID)
	if err != nil {
		logger.Error.Printf("Failed to get connected accounts: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get connected accounts"})
		return
	}

	result := make([]gin.H, 0, len(accounts))
	for _, acc := range accounts {
		result = append(result, gin.H{
			"id":         acc.ID,
			"provider":   acc.Provider,
			"email":      acc.Email,
			"created_at": acc.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"accounts": result})
}

func HandleGetAvailableCalendars(c *gin.Context) {
	user := getAuthenticatedUser(c)
	if user == nil {
		return
	}

	accountID := c.Param("id")

	account, err := database.GetConnectedAccountById(accountID)
	if err != nil {
		logger.Error.Printf("Failed to get connected account: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get account"})
		return
	}

	if account == nil || account.UserID != user.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	if calendarService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Calendar service not configured"})
		return
	}

	// Fetch calendars from Google
	googleCalendars, err := calendarService.ListCalendars(account.AccessToken, account.RefreshToken)
	if err != nil {
		logger.Error.Printf("Failed to fetch calendars from Google: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch calendars"})
		return
	}

	existingCalendars, err := database.GetCalendarsByUserId(user.ID)
	if err != nil {
		logger.Error.Printf("Failed to get existing calendars: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get calendars"})
		return
	}

	addedProviderIds := make(map[string]string)
	for _, cal := range existingCalendars {
		if cal.ConnectedAccountID != nil && *cal.ConnectedAccountID == accountID {
			addedProviderIds[cal.ProviderCalendarID] = cal.ID
		}
	}

	result := make([]gin.H, 0, len(googleCalendars))
	for _, cal := range googleCalendars {
		localID, added := addedProviderIds[cal.ProviderCalendarID]
		item := gin.H{
			"provider_calendar_id": cal.ProviderCalendarID,
			"name":                 cal.Name,
			"color":                cal.Color,
			"is_primary":           cal.IsPrimary,
			"added":                added,
			"local_id":             nil,
		}
		if added {
			item["local_id"] = localID
		}
		result = append(result, item)
	}

	c.JSON(http.StatusOK, gin.H{"calendars": result})
}

func HandleDeleteConnectedAccount(c *gin.Context) {
	user := getAuthenticatedUser(c)
	if user == nil {
		return
	}

	accountID := c.Param("id")

	account, err := database.GetConnectedAccountById(accountID)
	if err != nil {
		logger.Error.Printf("Failed to get connected account: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get account"})
		return
	}

	if account == nil || account.UserID != user.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	err = database.DeleteConnectedAccount(accountID)
	if err != nil {
		logger.Error.Printf("Failed to delete connected account: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete account"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}


func HandleGetCalendars(c *gin.Context) {
	user := getAuthenticatedUser(c)
	if user == nil {
		return
	}

	calendars, err := database.GetCalendarsByUserId(user.ID)
	if err != nil {
		logger.Error.Printf("Failed to get calendars: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get calendars"})
		return
	}

	result := make([]gin.H, 0, len(calendars))
	for _, cal := range calendars {
		webhookActive := cal.WebhookResourceID != nil && *cal.WebhookResourceID != ""
		result = append(result, gin.H{
			"id":                   cal.ID,
			"connected_account_id": cal.ConnectedAccountID,
			"provider_calendar_id": cal.ProviderCalendarID,
			"name":                 cal.Name,
			"color":                cal.Color,
			"is_primary":           cal.IsPrimary,
			"webhook_active":       webhookActive,
		})
	}

	c.JSON(http.StatusOK, gin.H{"calendars": result})
}

type AddCalendarRequest struct {
	ConnectedAccountID string `json:"connected_account_id" binding:"required"`
	ProviderCalendarID string `json:"provider_calendar_id" binding:"required"`
}

func HandleAddCalendar(c *gin.Context) {
	user := getAuthenticatedUser(c)
	if user == nil {
		return
	}

	var req AddCalendarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields"})
		return
	}

	account, err := database.GetConnectedAccountById(req.ConnectedAccountID)
	if err != nil {
		logger.Error.Printf("Failed to get connected account: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify account"})
		return
	}

	if account == nil || account.UserID != user.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Connected account not found"})
		return
	}

	existing, err := database.GetCalendarByProviderCalendarId(req.ConnectedAccountID, req.ProviderCalendarID)
	if err != nil {
		logger.Error.Printf("Failed to check existing calendar: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check calendar"})
		return
	}

	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Calendar already added"})
		return
	}

	if calendarService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Calendar service not configured"})
		return
	}

	googleCalendars, err := calendarService.ListCalendars(account.AccessToken, account.RefreshToken)
	if err != nil {
		logger.Error.Printf("Failed to fetch calendars from Google: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch calendar details"})
		return
	}

	var calendarToAdd *google.AvailableCalendar
	for _, cal := range googleCalendars {
		if cal.ProviderCalendarID == req.ProviderCalendarID {
			calendarToAdd = &cal
			break
		}
	}

	if calendarToAdd == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Calendar not found in Google"})
		return
	}

	calendarID, err := database.CreateCalendar(database.Calendar{
		UserID:             user.ID,
		ConnectedAccountID: &req.ConnectedAccountID,
		Provider:           "google",
		ProviderCalendarID: req.ProviderCalendarID,
		Name:               calendarToAdd.Name,
		Color:              calendarToAdd.Color,
		IsPrimary:          calendarToAdd.IsPrimary,
	})

	if err != nil {
		logger.Error.Printf("Failed to create calendar: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add calendar"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":      calendarID,
		"message": "Calendar added successfully",
	})
}

func HandleDeleteCalendar(c *gin.Context) {
	user := getAuthenticatedUser(c)
	if user == nil {
		return
	}

	calendarID := c.Param("id")

	calendar, err := database.GetCalendarById(calendarID)
	if err != nil {
		logger.Error.Printf("Failed to get calendar: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get calendar"})
		return
	}

	if calendar == nil || calendar.UserID != user.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Calendar not found"})
		return
	}

	err = database.DeleteCalendar(calendarID)
	if err != nil {
		logger.Error.Printf("Failed to delete calendar: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete calendar"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
