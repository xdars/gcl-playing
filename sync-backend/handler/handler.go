package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"calendar-backend/database"
	"shared/jwt"
	"shared/logger"
)

func HandleEvent(c *gin.Context) {
	eventId := c.Param("eventId")
	logger.Info.Printf("Received event: %s", eventId)

	c.JSON(http.StatusOK, gin.H{
		"message": "Event received",
		"eventId": eventId,
	})
}

func HandleAddCalendar(c *gin.Context) {
	authURL := "http://localhost:3000/auth"
	c.Redirect(http.StatusSeeOther, authURL)
}

func HandleTokens(c *gin.Context) {
	jwtCookie, err := c.Cookie("JWT")
	if err != nil {
		logger.Warn.Printf("JWT cookie not found: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sub, err := jwt.ParseJWT(jwtCookie)
	if err != nil {
		logger.Warn.Printf("Invalid JWT: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	user, err := database.GetUser(sub)
	if err != nil {
		logger.Error.Printf("Failed to get user %s: %v", sub, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  user.Token,
		"refresh_token": user.RToken,
	})
}
