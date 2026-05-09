package config

import (
	"log"
	"os"
	"regexp"

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
	CHAT_ROOMS string `yaml:"chat_rooms"`
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

// Regex to match ${VAR:-default}
var envVarWithDefaultRe = regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*):-([^}]*)\}`)

// expandEnvWithDefault replaces ${VAR:-default} patterns in the YAML file 
// with actual environment variables, falling back to the default if not set.
func expandEnvWithDefault(content []byte) []byte {
	// First expand ${VAR:-default}
	expanded := envVarWithDefaultRe.ReplaceAllFunc(content, func(match []byte) []byte {
		submatches := envVarWithDefaultRe.FindSubmatch(match)
		if len(submatches) == 3 {
			key := string(submatches[1])
			fallback := string(submatches[2])
			if val, exists := os.LookupEnv(key); exists && val != "" {
				return []byte(val)
			}
			return []byte(fallback)
		}
		return match
	})
	
	// Then expand regular ${VAR} or $VAR (if any exist without defaults)
	return []byte(os.ExpandEnv(string(expanded)))
}

// LoadConfig loads yaml configuration. Environment variables are injected into yaml directly.
func LoadConfig() (*Config, error) {
	// 1. Load .env file (if exists) into system ENV
	if err := godotenv.Load(); err != nil {
		log.Println("[CONFIG] No .env file found, using system environment variables")
	}

	cfg := &Config{}

	// 2. Read config.yaml
	yamlFile, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Println("[CONFIG] WARNING: No config.yaml found, app may fail if defaults aren't provided.")
		// Return empty struct so it doesn't crash, though it will lack defaults
		return cfg, nil
	}

	// 3. Expand ENV variables inside YAML content
	expandedYaml := expandEnvWithDefault(yamlFile)

	// 4. Parse YAML into struct
	if err := yaml.Unmarshal(expandedYaml, cfg); err != nil {
		log.Printf("[CONFIG] Error parsing config.yaml: %v\n", err)
	} else {
		log.Println("[CONFIG] Successfully loaded configuration from config.yaml (ENV merged)")
	}

	// Warn if using default secret
	if cfg.API_CONFIG.JWT_SECRET == "super-secret-key" {
		log.Println("[CONFIG] WARNING: Using default JWT_SECRET. Update .env for production.")
	}

	return cfg, nil
}
