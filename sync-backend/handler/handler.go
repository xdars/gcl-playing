package handler

import (
	"github.com/gin-gonic/gin"
	//"html/template"
	"log"
	"net/http"
	"calendar-backend/database"
	. "shared/jwt"
	//"io/ioutil"
)

func HandleEvent(c *gin.Context) {
	eventId := c.Param("eventId")
	log.Printf("Received event: %s", eventId)

	c.JSON(http.StatusOK, gin.H{
		"message": "Event received",
		"eventId": eventId,
	})
}

func HandleAddCalendar(c *gin.Context) {
	c.Redirect(http.StatusSeeOther, "http://localhost:3000/auth")
}

func HandleTokens(c *gin.Context) {
	jwt, err := c.Cookie("JWT")
	if err != nil {
		log.Println("JWT cookie not found:", err)
		c.JSON(http.StatusUnauthorized, gin.H{"msg": "unauthorized"})
		return
	}

	sub, err := ParseJWT(jwt)
	if err != nil {
		log.Println("Invalid JWT:", err)
		c.JSON(http.StatusUnauthorized, gin.H{"msg": "invalid token"})
		return
	}

	user, err := database.GetUser(sub)
	if err != nil {
		log.Println("Failed to get user:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "couldn't get user data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  user.Token,
		"refresh_token": user.RToken,
	})
}