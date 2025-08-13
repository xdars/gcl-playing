package main

import (
    "calendar-backend/config"
    "calendar-backend/router"
    "log"
)

func main() {
    cfg, err := config.LoadConfig("config.json")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    r := router.SetupRouter()
    log.Printf("Server running on port %s", cfg.Port)
    if err := r.Run(":" + cfg.Port); err != nil {
        log.Fatalf("Failed to run server: %v", err)
    }
}