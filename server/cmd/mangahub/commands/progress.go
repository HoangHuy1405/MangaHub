package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	cliclient "mangahub/internal/cliclient"
	"mangahub/pkg/models"
)

func NewProgressCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "progress",
		Short: "Reading progress management",
	}
	cmd.AddCommand(newProgressUpdateCmd(cfg))
	return cmd
}

func newProgressUpdateCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	var mangaID, device string
	var chapter int
	cmd := &cobra.Command{
		Use:     "update",
		Short:   "Update your reading progress for a manga",
		Example: "  mangahub progress update --manga-id one-piece --chapter 1095",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cfg.IsLoggedIn() {
				return fmt.Errorf("✗ Not logged in. Run: mangahub auth login --username <username>")
			}
			if mangaID == "" || chapter == 0 {
				return fmt.Errorf("both --manga-id and --chapter are required")
			}

			fmt.Println("Updating reading progress...")

			// 1. Update via REST API
			h := cliclient.NewHTTPClient(cfg)
			resp, err := h.Put("/users/progress", map[string]interface{}{
				"manga_id":        mangaID,
				"current_chapter": chapter,
			})
			if err != nil {
				return fmt.Errorf("✗ Progress update failed: %w", err)
			}
			if !resp.Success {
				return fmt.Errorf("✗ Progress update failed: %s", resp.Message)
			}

			fmt.Println("✓ Progress updated successfully!")
			fmt.Printf("  Manga:   %s\n", mangaID)
			fmt.Printf("  Chapter: %d\n\n", chapter)

			// 2. Try to sync via TCP — optional, non-blocking
			fmt.Println("Sync Status:")
			fmt.Println("  Local database: ✓ Updated")

			// Best Practice — Mistake #1: "Not Closing Connections"
			// Always defer Close() immediately after creating the client,
			// BEFORE attempting Connect(). Close() has a nil-check so it is
			// safe to call even if Connect() fails. This guarantees cleanup
			// on every code path (success, error, or panic).
			tcpClient := cliclient.NewTCPClient(cfg)
			defer tcpClient.Close()

			if err := tcpClient.Connect(cfg.Username); err != nil {
				fmt.Printf("  TCP sync server: ⚠ Not available (%v)\n", err)
				fmt.Println("  (Start TCP server to enable real-time sync across devices)")
			} else {
				update := models.ProgressUpdate{
					UserID:    cfg.Username,
					Device:    device,
					MangaID:   mangaID,
					Chapter:   chapter,
					Timestamp: time.Now().Unix(),
				}
				if err := tcpClient.SendProgress(update); err != nil {
					fmt.Printf("  TCP sync server: ⚠ Broadcast failed: %v\n", err)
				} else {
					fmt.Println("  TCP sync server: ✓ Broadcasting to connected devices")
				}
			}

			fmt.Printf("\nNext actions:\n")
			fmt.Printf("  Continue reading: mangahub manga info %s\n", mangaID)
			return nil
		},
	}
	cmd.Flags().StringVar(&mangaID, "manga-id", "", "Manga ID")
	cmd.Flags().IntVar(&chapter, "chapter", 0, "Chapter number")
	cmd.Flags().StringVar(&device, "device", "mobile", "Device name (for testing sync)")
	return cmd
}
