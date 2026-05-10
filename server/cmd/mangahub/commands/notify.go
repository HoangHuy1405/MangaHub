package commands

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	cliclient "mangahub/internal/cliclient"
)

func NewNotifyCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notify",
		Short: "UDP chapter-release notification subscription",
	}
	cmd.AddCommand(
		newNotifySubscribeCmd(cfg),
		newNotifyUnsubscribeCmd(cfg),
		newNotifyPreferencesCmd(cfg),
		newNotifyListenCmd(cfg),
		newNotifyTestCmd(cfg),
	)
	return cmd
}

// ── subscribe ─────────────────────────────────────────────────────────────────

func newNotifySubscribeCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	return &cobra.Command{
		Use:     "subscribe",
		Short:   "Subscribe this device to chapter-release notifications (UDP)",
		Example: "  mangahub notify subscribe",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cfg.IsLoggedIn() {
				return fmt.Errorf("✗ Not logged in. Run: mangahub auth login --username <username>")
			}
			fmt.Printf("Subscribing to UDP notifications at %s...\n", cfg.UDPAddr())

			u := cliclient.NewUDPClient(cfg)
			defer u.Close()

			if err := u.Register(cfg.Username); err != nil {
				return fmt.Errorf("✗ Subscription failed: %w\n\nMake sure the UDP server is running:\n  go build -o udp-server.exe ./cmd/udp-server && ./udp-server.exe", err)
			}

			fmt.Println("\n✓ Subscribed successfully!")
			fmt.Printf("  Server:  udp://%s\n", cfg.UDPAddr())
			fmt.Printf("  User:    %s\n", cfg.Username)
			fmt.Println("\nYou will now receive notifications when new chapters are released.")
			fmt.Println("Run 'mangahub notify listen' to watch for incoming notifications.")
			return nil
		},
	}
}

// ── unsubscribe ───────────────────────────────────────────────────────────────

func newNotifyUnsubscribeCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	return &cobra.Command{
		Use:     "unsubscribe",
		Short:   "Unsubscribe this device from chapter-release notifications",
		Example: "  mangahub notify unsubscribe",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cfg.IsLoggedIn() {
				return fmt.Errorf("✗ Not logged in")
			}
			u := cliclient.NewUDPClient(cfg)
			defer u.Close()

			if err := u.Unregister(cfg.Username); err != nil {
				return fmt.Errorf("✗ Unsubscribe failed: %w", err)
			}
			fmt.Println("✓ Unsubscribed from chapter notifications.")
			return nil
		},
	}
}

// ── preferences ───────────────────────────────────────────────────────────────

func newNotifyPreferencesCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	return &cobra.Command{
		Use:     "preferences",
		Short:   "View your notification preferences",
		Example: "  mangahub notify preferences",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cfg.IsLoggedIn() {
				return fmt.Errorf("✗ Not logged in")
			}
			fmt.Printf("Notification Preferences for %s:\n", cfg.Username)
			fmt.Println("  Email: enabled")
			fmt.Println("  In-app: enabled")
			fmt.Println("  Push (UDP): enabled")
			return nil
		},
	}
}

// ── listen ────────────────────────────────────────────────────────────────────

func newNotifyListenCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	return &cobra.Command{
		Use:     "listen",
		Short:   "Subscribe and listen for incoming chapter notifications (blocks)",
		Example: "  mangahub notify listen",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cfg.IsLoggedIn() {
				return fmt.Errorf("✗ Not logged in. Run: mangahub auth login --username <username>")
			}
			fmt.Printf("Subscribing to UDP notifications at %s...\n", cfg.UDPAddr())

			u := cliclient.NewUDPClient(cfg)
			defer u.Close()

			// Register first
			if err := u.Register(cfg.Username); err != nil {
				return fmt.Errorf("✗ Failed to register: %w", err)
			}

			// Best Practice — Mistake #6: "Goroutine Leaks"
			// Create a signal-aware context so Listen() can shut down
			// gracefully on Ctrl+C instead of leaking goroutines.
			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			// Then listen (blocks until context cancelled or Ctrl-C)
			return u.Listen(ctx)
		},
	}
}

// ── test ──────────────────────────────────────────────────────────────────────

func newNotifyTestCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	var mangaID string
	cmd := &cobra.Command{
		Use:     "test",
		Short:   "Send a test chapter notification via the REST API trigger",
		Example: "  mangahub notify test --manga-id one-piece",
		RunE: func(cmd *cobra.Command, args []string) error {
			if mangaID == "" {
				mangaID = "test-manga"
			}
			fmt.Printf("Sending test notification for manga '%s'...\n", mangaID)

			h := cliclient.NewHTTPClient(cfg)
			resp, err := h.Post("/notify/chapter", map[string]interface{}{
				"manga_id": mangaID,
				"message":  fmt.Sprintf("Test notification — Chapter X released at %s", time.Now().Format("15:04:05")),
			})
			if err != nil {
				return fmt.Errorf("✗ Notification trigger failed: %w", err)
			}
			if !resp.Success {
				return fmt.Errorf("✗ Server error: %s", resp.Message)
			}
			fmt.Println("✓ Test notification sent!")
			fmt.Println("  Any client running 'mangahub notify listen' should receive it now.")
			return nil
		},
	}
	cmd.Flags().StringVar(&mangaID, "manga-id", "", "Manga ID for the test notification")
	return cmd
}
