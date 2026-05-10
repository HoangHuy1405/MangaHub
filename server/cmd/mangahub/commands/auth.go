package commands

import (
	"encoding/json"
	"fmt"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	cliclient "mangahub/internal/cliclient"
)

func NewAuthCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication commands (register, login, logout, status)",
	}
	cmd.AddCommand(
		newRegisterCmd(cfg),
		newLoginCmd(cfg),
		newLogoutCmd(cfg),
		newAuthStatusCmd(cfg),
	)
	return cmd
}

// ── register ──────────────────────────────────────────────────────────────────

func newRegisterCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	var username string
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a new MangaHub account",
		Example: "  mangahub auth register --username johndoe",
		RunE: func(cmd *cobra.Command, args []string) error {
			if username == "" {
				return fmt.Errorf("--username is required")
			}
			fmt.Print("Password: ")
			pwBytes, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}
			password := strings.TrimSpace(string(pwBytes))
			if len(password) < 6 {
				return fmt.Errorf("✗ Registration failed: Password too weak\n  Password must be at least 6 characters")
			}

			h := cliclient.NewHTTPClient(cfg)
			resp, err := h.Post("/auth/register", map[string]string{
				"username": username,
				"password": password,
			})
			if err != nil {
				return fmt.Errorf("✗ Registration failed: %w", err)
			}
			if !resp.Success {
				return fmt.Errorf("✗ Registration failed: %s", resp.Message)
			}

			fmt.Printf("✓ Account created successfully!\n")
			fmt.Printf("  Username: %s\n", username)
			fmt.Printf("\nPlease login to start using MangaHub:\n")
			fmt.Printf("  mangahub auth login --username %s\n", username)
			return nil
		},
	}
	cmd.Flags().StringVar(&username, "username", "", "Username for the new account")
	return cmd
}

// ── login ─────────────────────────────────────────────────────────────────────

func newLoginCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	var username string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login to MangaHub and save your session token",
		Example: "  mangahub auth login --username johndoe",
		RunE: func(cmd *cobra.Command, args []string) error {
			if username == "" {
				return fmt.Errorf("--username is required")
			}
			fmt.Print("Password: ")
			pwBytes, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}

			h := cliclient.NewHTTPClient(cfg)
			resp, err := h.Post("/auth/login", map[string]string{
				"username": username,
				"password": strings.TrimSpace(string(pwBytes)),
			})
			if err != nil {
				return fmt.Errorf("✗ Login failed: %w", err)
			}
			if !resp.Success {
				return fmt.Errorf("✗ Login failed: %s", resp.Message)
			}

			// Extract token from response data
			var data struct {
				Token string `json:"token"`
			}
			if err := json.Unmarshal(resp.Data, &data); err != nil || data.Token == "" {
				return fmt.Errorf("✗ Login failed: server did not return a token")
			}

			cfg.Token = data.Token
			cfg.Username = username
			if err := cfg.Save(); err != nil {
				fmt.Printf("⚠ Warning: could not save session: %v\n", err)
			}

			fmt.Printf("✓ Login successful!\n")
			fmt.Printf("  Welcome back, %s!\n", username)
			fmt.Printf("  Auto-sync: enabled\n")
			fmt.Printf("  Notifications: enabled\n\n")
			fmt.Printf("Ready to use MangaHub! Try:\n")
			fmt.Printf("  mangahub manga search \"your favorite manga\"\n")
			return nil
		},
	}
	cmd.Flags().StringVar(&username, "username", "", "Your MangaHub username")
	return cmd
}

// ── logout ────────────────────────────────────────────────────────────────────

func newLogoutCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Logout and remove the stored session token",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.Token = ""
			cfg.Username = ""
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("could not clear session: %w", err)
			}
			fmt.Println("✓ Logged out successfully. Your session token has been removed.")
			return nil
		},
	}
}

// ── status ────────────────────────────────────────────────────────────────────

func newAuthStatusCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the current authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cfg.IsLoggedIn() {
				fmt.Println("Status: ✗ Not logged in")
				fmt.Println("Run: mangahub auth login --username <your-username>")
				return nil
			}
			fmt.Printf("Status:   ✓ Logged in\n")
			fmt.Printf("Username: %s\n", cfg.Username)
			fmt.Printf("Server:   %s\n", cfg.HTTPURL())
			return nil
		},
	}
}
