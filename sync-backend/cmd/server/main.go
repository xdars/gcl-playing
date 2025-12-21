package main

import (
	"log"

	"calendar-backend/config"
	"calendar-backend/router"
)

func main() {
	cfg := config.LoadConfig()

	r := router.SetupRouter()
	log.Printf("Server running on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
