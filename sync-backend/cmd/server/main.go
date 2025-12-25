package main

import (
	"log"

	"github.com/joho/godotenv"

	"calendar-backend/config"
	"calendar-backend/handler"
	"calendar-backend/router"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found, using environment variables")
	}

	cfg := config.LoadConfig()

	handler.InitCalendarService()

	r := router.SetupRouter()
	log.Printf("Server running on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
