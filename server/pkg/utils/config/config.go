package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type APIConfig struct {
	API_PORT                  string `yaml:"api_port"`
	DB_PATH                   string `yaml:"db_path"`
	JWT_SECRET                string `yaml:"jwt_secret"`
	JWT_ACCESS_TOKEN_LIFETIME string `yaml:"jwt_access_token_lifetime"`
}

type SocketConfig struct {
	CHAT_ROOMS []string `yaml:"chat_rooms"`
}

type NetworkConfig struct {
	TCP_PORT string `yaml:"tcp_port"`
	UDP_PORT string `yaml:"udp_port"`
}

type Config struct {
	API_CONFIG     APIConfig     `yaml:"api"`
	SOCKET_CONFIG  SocketConfig  `yaml:"socket"`
	NETWORK_CONFIG NetworkConfig `yaml:"network"`
}

// LoadConfig loads yaml configuration and then applies environment variable overrides.
func LoadConfig() (*Config, error) {
	// 1. Load .env file (if exists) into system ENV
	if err := godotenv.Load(); err != nil {
		log.Println("[CONFIG] No .env file found, using system environment variables")
	}

	cfg := &Config{}

	// 2. Read config.yaml (Base config)
	yamlFile, err := os.ReadFile("config.yaml")
	if err == nil {
		if err := yaml.Unmarshal(yamlFile, cfg); err != nil {
			log.Printf("[CONFIG] Error parsing config.yaml: %v\n", err)
		} else {
			log.Println("[CONFIG] Loaded base configuration from config.yaml")
		}
	} else {
		log.Println("[CONFIG] No config.yaml found, relying entirely on ENV / hardcoded defaults")
	}

	// Helper to fallback to YAML -> then hardcoded default
	fallbackStr := func(yamlVal, defaultVal string) string {
		if yamlVal != "" {
			return yamlVal
		}
		return defaultVal
	}
	
	fallbackList := func(yamlVal []string, defaultVal string) []string {
		if len(yamlVal) > 0 {
			return yamlVal
		}
		return []string{defaultVal}
	}

	// 3. Apply ENV overrides (ENV wins over YAML)
	cfg.API_CONFIG.API_PORT = getEnvAsStr("API_PORT", fallbackStr(cfg.API_CONFIG.API_PORT, "8080"))
	cfg.API_CONFIG.DB_PATH = getEnvAsStr("DB_PATH", fallbackStr(cfg.API_CONFIG.DB_PATH, "data/mangahub.db"))
	cfg.API_CONFIG.JWT_SECRET = getEnvAsStr("JWT_SECRET", fallbackStr(cfg.API_CONFIG.JWT_SECRET, "super-secret-key"))
	cfg.API_CONFIG.JWT_ACCESS_TOKEN_LIFETIME = getEnvAsStr("JWT_ACCESS_TOKEN_LIFETIME", fallbackStr(cfg.API_CONFIG.JWT_ACCESS_TOKEN_LIFETIME, "24h"))
	
	cfg.SOCKET_CONFIG.CHAT_ROOMS = getEnvAsListStr("CHAT_ROOMS", fallbackList(cfg.SOCKET_CONFIG.CHAT_ROOMS, "general"))
	
	cfg.NETWORK_CONFIG.TCP_PORT = getEnvAsStr("TCP_PORT", fallbackStr(cfg.NETWORK_CONFIG.TCP_PORT, "9090"))
	cfg.NETWORK_CONFIG.UDP_PORT = getEnvAsStr("UDP_PORT", fallbackStr(cfg.NETWORK_CONFIG.UDP_PORT, "9091"))


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

func getEnvAsListStr(key string, fallback []string) []string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return strings.Split(value, ",")
	}
	return fallback
}
