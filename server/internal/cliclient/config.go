package cliclient

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const configFileName = "config.json"

// CLIConfig stores the CLI client's runtime settings.
// It is persisted to ~/.mangahub/config.json between sessions.
type CLIConfig struct {
	ServerHost  string `json:"server_host"`
	HTTPPort    string `json:"http_port"`
	TCPPort     string `json:"tcp_port"`
	UDPPort     string `json:"udp_port"`
	Token       string `json:"token"`
	Username    string `json:"username"`
}

// Default returns a CLIConfig with sensible defaults that match config.yaml.
func Default() *CLIConfig {
	return &CLIConfig{
		ServerHost: "localhost",
		HTTPPort:   "8080",
		TCPPort:    "9090",
		UDPPort:    "9091",
	}
}

// configDir returns ~/.mangahub
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".mangahub"), nil
}

// Load reads the persisted CLI config from ~/.mangahub/config.json.
// If the file doesn't exist, it returns Default() and creates the directory.
func Load() (*CLIConfig, error) {
	dir, err := configDir()
	if err != nil {
		return Default(), nil
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return Default(), nil
	}

	path := filepath.Join(dir, configFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		// First run — return defaults, no error
		return Default(), nil
	}

	cfg := Default()
	if err := json.Unmarshal(data, cfg); err != nil {
		return Default(), fmt.Errorf("corrupt config file: %w", err)
	}
	return cfg, nil
}

// Save writes the CLI config to ~/.mangahub/config.json.
func (c *CLIConfig) Save() error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, configFileName), data, 0600)
}

// HTTPURL returns the base URL for the REST API.
func (c *CLIConfig) HTTPURL() string {
	return fmt.Sprintf("http://%s:%s/api/v1", c.ServerHost, c.HTTPPort)
}

// TCPAddr returns the TCP server address string.
func (c *CLIConfig) TCPAddr() string {
	return fmt.Sprintf("%s:%s", c.ServerHost, c.TCPPort)
}

// UDPAddr returns the UDP server address string.
func (c *CLIConfig) UDPAddr() string {
	return fmt.Sprintf("%s:%s", c.ServerHost, c.UDPPort)
}

// IsLoggedIn returns true when a JWT token is persisted.
func (c *CLIConfig) IsLoggedIn() bool {
	return c.Token != ""
}
