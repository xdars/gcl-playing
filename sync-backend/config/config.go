package config

import (
	"os"
	"strings"
)

type Config struct {
	Port           string
	AllowedOrigins []string
	DatabasePath   string
	Environment    string
	FrontendDir    string
}

var Cfg *Config

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func LoadConfig() *Config {
	Cfg = &Config{
		Port:           getEnv("SYNC_BACKEND_PORT", "8080"),
		AllowedOrigins: strings.Split(getEnv("ALLOWED_ORIGINS", "http://localhost:5173"), ","),
		DatabasePath:   getEnv("DATABASE_PATH", "./data.db"),
		Environment:    getEnv("GO_ENV", "development"),
		FrontendDir:    getEnv("FRONTEND_DIR", "../frontend/dist"),
	}
	return Cfg
}
