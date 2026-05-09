package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type APIConfig struct {
	API_PORT                  string
	DB_PATH                   string
	JWT_SECRET                string
	JWT_ACCESS_TOKEN_LIFETIME string
}

type SocketConfig struct {
	CHAT_ROOMS []string
}

type Config struct {
	API_CONFIG    APIConfig
	SOCKET_CONFIG SocketConfig
}

// LoadConfig loads environment variables and returns a Config struct.
func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("[CONFIG] No .env file found, using system environment variables")
	}

	cfg := &Config{
		API_CONFIG: APIConfig{
			API_PORT:                  getEnvAsStr("API_PORT", "8080"),
			DB_PATH:                   getEnvAsStr("DB_PATH", "data/mangahub.db"),
			JWT_SECRET:                getEnvAsStr("JWT_SECRET", "super-secret-key"),
			JWT_ACCESS_TOKEN_LIFETIME: getEnvAsStr("JWT_ACCESS_TOKEN_LIFETIME", "24h"),
		},
		SOCKET_CONFIG: SocketConfig{
			CHAT_ROOMS: getEnvAsListStr("CHAT_ROOMS", "general"),
		},
	}

	// Warn if using default secret
	if cfg.API_CONFIG.JWT_SECRET == "super-secret-key" {
		log.Println("[CONFIG] WARNING: Using default JWT_SECRET. Update .env for production.")
	}

	return cfg, nil
}

func getEnvAsStr(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return fallback
}

func getEnvAsListStr(key, fallback string) []string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return strings.Split(value, ",")
	}
	return []string{fallback}
}
