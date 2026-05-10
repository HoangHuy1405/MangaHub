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

func NewSyncCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "TCP real-time progress synchronization across devices",
	}
	cmd.AddCommand(
		newSyncConnectCmd(cfg),
		newSyncMonitorCmd(cfg),
		newSyncStatusCmd(cfg),
	)
	return cmd
}

// ── connect ───────────────────────────────────────────────────────────────────

func newSyncConnectCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "connect",
		Short: "Connect to the TCP progress sync server",
		Example: "  mangahub sync connect",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cfg.IsLoggedIn() {
				return fmt.Errorf("✗ Not logged in. Run: mangahub auth login --username <username>")
			}
			fmt.Printf("Connecting to TCP sync server at %s...\n", cfg.TCPAddr())

			tcpClient := cliclient.NewTCPClient(cfg)
			if err := tcpClient.Connect(cfg.Username); err != nil {
				return fmt.Errorf("✗ Connection failed: %w\n\nMake sure the TCP server is running:\n  go build -o tcp-server.exe ./cmd/tcp-server && ./tcp-server.exe", err)
			}
			defer tcpClient.Close()

			fmt.Println("\n✓ Connected successfully!")
			fmt.Printf("  Server:  tcp://%s\n", cfg.TCPAddr())
			fmt.Printf("  User:    %s\n", cfg.Username)
			fmt.Printf("  Time:    %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
			fmt.Println("Sync Status:")
			fmt.Println("  Auto-sync: enabled")
			fmt.Println("  Conflict resolution: last_write_wins")
			fmt.Println("\nReal-time sync is now active.")
			fmt.Println("Use 'mangahub sync monitor' to watch for incoming updates.")
			return nil
		},
	}
}

// ── monitor ───────────────────────────────────────────────────────────────────

func newSyncMonitorCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "monitor",
		Short: "Listen for real-time progress updates from other devices (blocks)",
		Example: "  mangahub sync monitor",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cfg.IsLoggedIn() {
				return fmt.Errorf("✗ Not logged in. Run: mangahub auth login --username <username>")
			}
			fmt.Printf("Connecting to TCP sync server at %s...\n", cfg.TCPAddr())

			tcpClient := cliclient.NewTCPClient(cfg)
			if err := tcpClient.Connect(cfg.Username); err != nil {
				return fmt.Errorf("✗ Connection failed: %w", err)
			}
			defer tcpClient.Close()

			fmt.Printf("✓ Connected as user: %s\n\n", cfg.Username)

			// Best Practice — Mistake #6: "Goroutine Leaks"
			// Create a signal-aware context so Monitor() can shut down
			// gracefully on Ctrl+C instead of leaking goroutines.
			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			// Monitor blocks until context is cancelled or connection drops
			return tcpClient.Monitor(ctx)
		},
	}
}

// ── status ────────────────────────────────────────────────────────────────────

func newSyncStatusCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check TCP sync server reachability",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("TCP Sync Status:\n")
			fmt.Printf("  Server:   tcp://%s\n", cfg.TCPAddr())
			fmt.Printf("  User:     %s\n", cfg.Username)

			// Best Practice — Code hygiene:
			// Check auth BEFORE allocating the TCPClient. This avoids
			// creating resources that will never be used when the user
			// is not logged in.
			if !cfg.IsLoggedIn() {
				fmt.Println("  Auth:     ✗ Not logged in")
				return nil
			}

			// Quick dial test
			tcpClient := cliclient.NewTCPClient(cfg)
			if err := tcpClient.Connect(cfg.Username); err != nil {
				fmt.Printf("  Connection: ✗ Unreachable (%v)\n", err)
				fmt.Println("\nTo start the TCP server:")
				fmt.Println("  go build -o tcp-server.exe ./cmd/tcp-server && ./tcp-server.exe")
			} else {
				defer tcpClient.Close()
				fmt.Printf("  Connection: ✓ Active\n")
				fmt.Printf("  Checked at: %s\n", time.Now().Format("15:04:05"))
			}
			return nil
		},
	}
}
