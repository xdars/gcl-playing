package router

import (
	"os"
	"path/filepath"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"calendar-backend/config"
	"calendar-backend/handler"
	"shared/middleware"
)

func SetupRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(middleware.Recovery())

	cfg := config.Cfg

	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	distDir := cfg.FrontendDir

	r.POST("/event/:eventId", handler.HandleEvent)
	r.Static("/home/assets", filepath.Join(distDir, "assets"))
	r.StaticFile("/home/favicon.ico", filepath.Join(distDir, "favicon.ico"))

	r.GET("/home", serveIndex(distDir))
	r.GET("/auth/callback", handler.HandleAuthCallback)

	// API routes
	r.GET("/api/tokens", handler.HandleTokens)

	// Connected accounts
	r.GET("/api/connected-accounts", handler.HandleGetConnectedAccounts)
	r.GET("/api/connected-accounts/:id/calendars", handler.HandleGetAvailableCalendars)
	r.DELETE("/api/connected-accounts/:id", handler.HandleDeleteConnectedAccount)

	// Calendars
	r.GET("/api/calendars", handler.HandleGetCalendars)
	r.POST("/api/calendars", handler.HandleAddCalendar)
	r.DELETE("/api/calendars/:id", handler.HandleDeleteCalendar)

	return r
}

func serveIndex(distDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		indexPath := filepath.Join(distDir, "index.html")
		if _, err := os.Stat(indexPath); err != nil {
			c.String(500, "index.html not found: "+err.Error())
			return
		}
		c.File(indexPath)
	}
}
