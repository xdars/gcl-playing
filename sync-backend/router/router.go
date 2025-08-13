package router

import (
	"calendar-backend/handler"
	"github.com/gin-gonic/gin"
    "path/filepath"
    "os"
    "github.com/gin-contrib/cors"
    "time"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

    r.Use(cors.New(cors.Config{
        AllowOrigins:     []string{"http://localhost:5173"},
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "Access-Control-Allow-Credentials"},
        ExposeHeaders:    []string{"Content-Length"},
        AllowCredentials: true,
        MaxAge: 12 * time.Hour,
    }))

    distDir := "../frontend/dist"

	r.LoadHTMLGlob("templates/*.tmpl")
	r.POST("/event/:eventId", handler.HandleEvent)
	r.Static("/home/assets", filepath.Join(distDir, "assets"))
	r.StaticFile("/home/favicon.ico", filepath.Join(distDir, "favicon.ico"))

	r.GET("/home", serveIndex(distDir))
	r.GET("/api/tokens", handler.HandleTokens)
  //r.GET("/addcalendar", handler.HandleAddCalender)
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