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

func NewChatCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chat",
		Short: "WebSocket real-time manga chat",
	}
	cmd.AddCommand(
		newChatJoinCmd(cfg),
		newChatSendCmd(cfg),
		newChatStatusCmd(cfg),
		newChatHistoryCmd(cfg),
	)
	return cmd
}

// ── join ──────────────────────────────────────────────────────────────────────

func newChatJoinCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	var mangaID string
	cmd := &cobra.Command{
		Use:   "join",
		Short: "Join a chat room (interactive mode)",
		Long: `Join a WebSocket chat room in interactive mode.
Without --manga-id, joins the general discussion room.
With --manga-id, joins a manga-specific discussion room.`,
		Example: `  mangahub chat join
  mangahub chat join --manga-id one-piece`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// ── Authorization: only authenticated users may join chat ─────
			if !cfg.IsLoggedIn() {
				return fmt.Errorf("✗ Not logged in. Run: mangahub auth login --username <username>")
			}

			room := "general"
			if mangaID != "" {
				room = mangaID
			}

			fmt.Printf("Connecting to WebSocket chat server at %s...\n", cfg.WSURL())

			wsClient := cliclient.NewWSClient(cfg)
			if err := wsClient.Connect(cfg.Username, room); err != nil {
				return fmt.Errorf("✗ Connection failed: %w\n\nMake sure the API server is running:\n  go build -o api-server.exe ./cmd/api-server && .\\api-server.exe", err)
			}

			// Best Practice — Mistake #6: "Goroutine Leaks"
			// Create a signal-aware context so Listen() can shut down
			// gracefully on Ctrl+C instead of leaking goroutines.
			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			// Listen blocks until /quit, Ctrl+C, or connection drops.
			return wsClient.Listen(ctx)
		},
	}
	cmd.Flags().StringVar(&mangaID, "manga-id", "", "Join a manga-specific discussion room")
	return cmd
}

// ── send ──────────────────────────────────────────────────────────────────────

func newChatSendCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	var mangaID string
	cmd := &cobra.Command{
		Use:   "send [message]",
		Short: "Send a one-shot message to a chat room",
		Long: `Connect to the chat server, send a single message, and disconnect.
The message is broadcast to all users in the target room.`,
		Example: `  mangahub chat send "Hello everyone!"
  mangahub chat send "Great chapter!" --manga-id one-piece`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cfg.IsLoggedIn() {
				return fmt.Errorf("✗ Not logged in. Run: mangahub auth login --username <username>")
			}

			room := "general"
			if mangaID != "" {
				room = mangaID
			}
			message := args[0]

			fmt.Printf("Connecting to chat server...\n")

			wsClient := cliclient.NewWSClient(cfg)
			if err := wsClient.Connect(cfg.Username, room); err != nil {
				return fmt.Errorf("✗ Connection failed: %w", err)
			}
			defer wsClient.Close()

			if err := wsClient.SendMessage(message); err != nil {
				return fmt.Errorf("✗ Send failed: %w", err)
			}

			// Brief wait for server to process and broadcast.
			time.Sleep(500 * time.Millisecond)

			fmt.Printf("✓ Message sent to #%s: \"%s\"\n", room, message)
			return nil
		},
	}
	cmd.Flags().StringVar(&mangaID, "manga-id", "", "Target manga-specific chat room")
	return cmd
}

// ── status ────────────────────────────────────────────────────────────────────

func newChatStatusCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	return &cobra.Command{
		Use:     "status",
		Short:   "Check WebSocket chat server reachability",
		Example: "  mangahub chat status",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("WebSocket Chat Status:\n")
			fmt.Printf("  Server:   %s\n", cfg.WSURL())
			fmt.Printf("  User:     %s\n", cfg.Username)

			if !cfg.IsLoggedIn() {
				fmt.Println("  Auth:     ✗ Not logged in")
				return nil
			}

			// Quick dial test.
			wsClient := cliclient.NewWSClient(cfg)
			if err := wsClient.Connect(cfg.Username, "general"); err != nil {
				fmt.Printf("  Connection: ✗ Unreachable (%v)\n", err)
				fmt.Println("\nTo start the server:")
				fmt.Println("  go build -o api-server.exe ./cmd/api-server && .\\api-server.exe")
			} else {
				defer wsClient.Close()
				fmt.Printf("  Connection: ✓ Active\n")
				fmt.Printf("  Checked at: %s\n", time.Now().Format("15:04:05"))
			}
			return nil
		},
	}
}

// ── history ───────────────────────────────────────────────────────────────────

func newChatHistoryCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	var mangaID string
	var limit int
	cmd := &cobra.Command{
		Use:   "history",
		Short: "View recent chat messages",
		Long:  `Connect to the chat server and request the recent message history for a room.`,
		Example: `  mangahub chat history
  mangahub chat history --manga-id one-piece --limit 50`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cfg.IsLoggedIn() {
				return fmt.Errorf("✗ Not logged in. Run: mangahub auth login --username <username>")
			}

			room := "general"
			if mangaID != "" {
				room = mangaID
			}
			_ = limit // limit is for future server-side pagination

			fmt.Printf("Fetching chat history for #%s...\n", room)

			wsClient := cliclient.NewWSClient(cfg)
			if err := wsClient.Connect(cfg.Username, room); err != nil {
				return fmt.Errorf("✗ Connection failed: %w", err)
			}
			defer wsClient.Close()

			// Request history and wait briefly for response.
			if err := wsClient.SendMessage("/history-request"); err != nil {
				return fmt.Errorf("✗ Failed to request history: %w", err)
			}

			// Give the server time to respond before disconnecting.
			time.Sleep(1 * time.Second)

			return nil
		},
	}
	cmd.Flags().StringVar(&mangaID, "manga-id", "", "View history for a manga-specific room")
	cmd.Flags().IntVar(&limit, "limit", 20, "Number of recent messages to show")
	return cmd
}
