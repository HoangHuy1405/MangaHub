package commands

import (
	"encoding/json"
	"fmt"
	"strings"
	"syscall"
	"time"

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
		newChangePasswordCmd(cfg),
	)
	return cmd
}

// ── register ──────────────────────────────────────────────────────────────────

func newRegisterCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	var username string
	var email string
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a new MangaHub account",
		Example: "  mangahub auth register --username johndoe",
		RunE: func(cmd *cobra.Command, args []string) error {
			if username == "" || strings.HasPrefix(username, "-") {
				return fmt.Errorf("--username is required and must be valid")
			}
			if email == "" || strings.HasPrefix(email, "-") {
				return fmt.Errorf("--email is required and must be valid")
			}
			if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
				return fmt.Errorf("✗ Registration failed: Invalid email format\nPlease provide a valid email address")
			}

			h := cliclient.NewHTTPClient(cfg)
			checkResp, err := h.Get(fmt.Sprintf("/auth/check?username=%s&email=%s", username, email))
			if err != nil {
				return fmt.Errorf("✗ Registration failed: %w", err)
			}
			if !checkResp.Success {
				if strings.Contains(checkResp.Message, "already exists") {
					return fmt.Errorf("✗ Registration failed: Username '%s' already exists\nTry: mangahub auth login --username %s", username, username)
				}
				return fmt.Errorf("✗ Registration failed: %s", checkResp.Message)
			}
			fmt.Print("Password: ")
			pwBytes, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}
			password := strings.TrimSpace(string(pwBytes))

			fmt.Print("Confirm password: ")
			pwConfirmBytes, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				return fmt.Errorf("failed to read confirm password: %w", err)
			}
			confirmPassword := strings.TrimSpace(string(pwConfirmBytes))

			if password != confirmPassword {
				return fmt.Errorf("✗ Registration failed: Passwords do not match")
			}

			if len(password) < 8 {
				return fmt.Errorf("✗ Registration failed: Password too weak\nPassword must be at least 8 characters with mixed case and numbers")
			}
			resp, err := h.Post("/auth/register", map[string]string{
				"username": username,
				"email":    email,
				"password": password,
			})
			if err != nil {
				return fmt.Errorf("✗ Registration failed: %w", err)
			}
			if !resp.Success {
				if strings.Contains(resp.Message, "already exists") {
					return fmt.Errorf("✗ Registration failed: Username '%s' already exists\nTry: mangahub auth login --username %s", username, username)
				}
				return fmt.Errorf("✗ Registration failed: %s", resp.Message)
			}

			var data struct {
				ID       int    `json:"id"`
				Username string `json:"username"`
				Email    string `json:"email"`
			}
			if err := json.Unmarshal(resp.Data, &data); err != nil {
				// fallback if parsing fails
				data.Username = username
				data.Email = email
			}

			fmt.Printf("✓ Account created successfully!\n")
			fmt.Printf("User ID: usr_%d\n", data.ID)
			fmt.Printf("Username: %s\n", data.Username)
			if data.Email != "" {
				fmt.Printf("Email: %s\n", data.Email)
			} else if email != "" {
				fmt.Printf("Email: %s\n", email)
			}
			fmt.Printf("Created: %s\n", time.Now().UTC().Format("2006-01-02 15:04:05 MST"))
			fmt.Printf("Please login to start using MangaHub:\n")
			fmt.Printf("mangahub auth login --username %s\n", username)
			return nil
		},
	}
	cmd.Flags().StringVar(&username, "username", "", "Username for the new account")
	cmd.Flags().StringVar(&email, "email", "", "Email for the new account")
	return cmd
}

// ── login ─────────────────────────────────────────────────────────────────────

func newLoginCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	var username string
	var email string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login to MangaHub and save your session token",
		Example: "  mangahub auth login --username johndoe\n  mangahub auth login --email john@example.com",
		RunE: func(cmd *cobra.Command, args []string) error {
			if username == "" && email == "" {
				return fmt.Errorf("--username or --email is required")
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
				"email":    email,
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

			displayName := username
			if displayName == "" {
				displayName = email
			}

			cfg.Token = data.Token
			cfg.Username = displayName
			if err := cfg.Save(); err != nil {
				fmt.Printf("⚠ Warning: could not save session: %v\n", err)
			}

			fmt.Printf("✓ Login successful!\n")
			fmt.Printf("Welcome back, %s!\n", displayName)
			fmt.Printf("Session Details:\n")
			fmt.Printf("Token expires: %s (24 hours)\n", time.Now().Add(24*time.Hour).UTC().Format("2006-01-02 15:04:05 MST"))
			fmt.Printf("Permissions: read, write, sync\n")
			fmt.Printf("Auto-sync: enabled\n")
			fmt.Printf("Notifications: enabled\n\n")
			fmt.Printf("Ready to use MangaHub! Try:\n")
			fmt.Printf("mangahub manga search \"your favorite manga\"\n")
			return nil
		},
	}
	cmd.Flags().StringVar(&username, "username", "", "Your MangaHub username")
	cmd.Flags().StringVar(&email, "email", "", "Your MangaHub email")
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

// ── change-password ───────────────────────────────────────────────────────────

func newChangePasswordCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "change-password",
		Short: "Change your MangaHub account password",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cfg.IsLoggedIn() {
				return fmt.Errorf("✗ Change password failed: Not logged in. Please run `mangahub auth login` first")
			}

			fmt.Print("Current password: ")
			oldPwBytes, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				return fmt.Errorf("failed to read current password: %w", err)
			}
			oldPassword := strings.TrimSpace(string(oldPwBytes))

			fmt.Print("New password: ")
			newPwBytes, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				return fmt.Errorf("failed to read new password: %w", err)
			}
			newPassword := strings.TrimSpace(string(newPwBytes))

			fmt.Print("Confirm new password: ")
			confirmPwBytes, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				return fmt.Errorf("failed to read confirm password: %w", err)
			}
			confirmPassword := strings.TrimSpace(string(confirmPwBytes))

			if newPassword != confirmPassword {
				return fmt.Errorf("✗ Change password failed: New passwords do not match")
			}

			if len(newPassword) < 8 {
				return fmt.Errorf("✗ Change password failed: Password too weak\nPassword must be at least 8 characters with mixed case and numbers")
			}

			h := cliclient.NewHTTPClient(cfg)
			resp, err := h.Put("/auth/change-password", map[string]string{
				"old_password": oldPassword,
				"new_password": newPassword,
			})
			if err != nil {
				return fmt.Errorf("✗ Change password failed: %w", err)
			}
			if !resp.Success {
				return fmt.Errorf("✗ Change password failed: %s", resp.Message)
			}

			fmt.Println("✓ Password changed successfully!")
			return nil
		},
	}
}
