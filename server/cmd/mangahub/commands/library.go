package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	cliclient "mangahub/internal/cliclient"
)

func NewLibraryCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "library",
		Short: "Manage your personal manga library",
	}
	cmd.AddCommand(
		newLibraryAddCmd(cfg),
		newLibraryListCmd(cfg),
		newLibraryRemoveCmd(cfg),
	)
	return cmd
}

// ── add ───────────────────────────────────────────────────────────────────────

func newLibraryAddCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	var mangaID, status string
	cmd := &cobra.Command{
		Use:     "add",
		Short:   "Add a manga to your library",
		Example: "  mangahub library add --manga-id one-piece --status reading",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cfg.IsLoggedIn() {
				return fmt.Errorf("✗ Not logged in. Run: mangahub auth login --username <username>")
			}
			if mangaID == "" {
				return fmt.Errorf("--manga-id is required")
			}
			if status == "" {
				status = "plan_to_read"
			}
			h := cliclient.NewHTTPClient(cfg)
			resp, err := h.Post("/users/library", map[string]string{
				"manga_id": mangaID,
				"status":   status,
			})
			if err != nil {
				return err
			}
			if !resp.Success {
				return fmt.Errorf("✗ Failed to add manga: %s", resp.Message)
			}
			fmt.Printf("✓ Added '%s' to your library with status: %s\n", mangaID, status)
			fmt.Printf("  Update progress: mangahub progress update --manga-id %s --chapter 1\n", mangaID)
			return nil
		},
	}
	cmd.Flags().StringVar(&mangaID, "manga-id", "", "Manga ID to add")
	cmd.Flags().StringVar(&status, "status", "plan_to_read", "Reading status (reading/completed/plan_to_read)")
	return cmd
}

// ── list ──────────────────────────────────────────────────────────────────────

func newLibraryListCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	var status string
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "View your manga library",
		Example: "  mangahub library list\n  mangahub library list --status reading",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cfg.IsLoggedIn() {
				return fmt.Errorf("✗ Not logged in. Run: mangahub auth login --username <username>")
			}
			path := "/users/library"
			if status != "" {
				path += "?status=" + status
			}
			h := cliclient.NewHTTPClient(cfg)
			resp, err := h.Get(path)
			if err != nil {
				return err
			}
			if !resp.Success {
				return fmt.Errorf("failed to fetch library: %s", resp.Message)
			}

			var library []struct {
				MangaID        string `json:"manga_id"`
				CurrentChapter int    `json:"current_chapter"`
				Status         string `json:"status"`
				UpdatedAt      string `json:"updated_at"`
			}
			if err := json.Unmarshal(resp.Data, &library); err != nil {
				return fmt.Errorf("parse error: %w", err)
			}

			if len(library) == 0 {
				fmt.Println("Your library is empty.")
				fmt.Println("Get started:")
				fmt.Println("  mangahub manga search \"your favorite series\"")
				fmt.Println("  mangahub library add --manga-id <id> --status reading")
				return nil
			}

			fmt.Printf("Your Manga Library (%d entries)\n\n", len(library))
			fmt.Printf("%-28s %-10s %-15s %s\n", "Manga ID", "Chapter", "Status", "Updated")
			fmt.Println(strings.Repeat("─", 75))
			for _, e := range library {
				updated := e.UpdatedAt
				if len(updated) > 10 {
					updated = updated[:10]
				}
				fmt.Printf("%-28s %-10d %-15s %s\n",
					e.MangaID, e.CurrentChapter, e.Status, updated)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "Filter by status")
	return cmd
}

// ── remove ────────────────────────────────────────────────────────────────────

func newLibraryRemoveCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	var mangaID string
	cmd := &cobra.Command{
		Use:     "remove",
		Short:   "Remove a manga from your library",
		Example: "  mangahub library remove --manga-id one-piece",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cfg.IsLoggedIn() {
				return fmt.Errorf("✗ Not logged in. Run: mangahub auth login --username <username>")
			}
			if mangaID == "" {
				return fmt.Errorf("--manga-id is required")
			}
			h := cliclient.NewHTTPClient(cfg)
			resp, err := h.Delete("/users/library/" + mangaID)
			if err != nil {
				return err
			}
			if !resp.Success {
				return fmt.Errorf("✗ Failed: %s", resp.Message)
			}
			fmt.Printf("✓ Removed '%s' from your library.\n", mangaID)
			return nil
		},
	}
	cmd.Flags().StringVar(&mangaID, "manga-id", "", "Manga ID to remove")
	return cmd
}
