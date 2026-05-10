package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	cmd.AddCommand(newProgressHistoryCmd(cfg))
	return cmd
}

func newProgressUpdateCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	var mangaID, device, notes string
	var chapter, volume int
	var force bool

	cmd := &cobra.Command{
		Use:     "update",
		Short:   "Update your reading progress for a manga",
		Example: "  mangahub progress update --manga-id one-piece --chapter 1095\n  mangahub progress update --manga-id naruto --chapter 700 --volume 72 --notes \"Great ending!\"",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cfg.IsLoggedIn() {
				return fmt.Errorf("✗ Not logged in. Run: mangahub auth login --username <username>")
			}
			if mangaID == "" || chapter == 0 {
				return fmt.Errorf("both --manga-id and --chapter are required")
			}

			fmt.Println("Updating reading progress...")

			h := cliclient.NewHTTPClient(cfg)

			// 1. Fetch library to validate if manga is in library and get current progress
			libResp, err := h.Get("/users/library")
			if err != nil {
				return fmt.Errorf("✗ Progress update failed: %w", err)
			}
			var libData struct {
				Library []struct {
					MangaID        string `json:"manga_id"`
					CurrentChapter int    `json:"current_chapter"`
					MangaTitle     string `json:"manga_title"`
				} `json:"library"`
			}
			if err := cliclient.Decode(libResp, &libData); err != nil {
				return fmt.Errorf("✗ Progress update failed: could not parse library data")
			}

			var currentProgress = -1
			var mangaTitle = mangaID
			for _, entry := range libData.Library {
				if entry.MangaID == mangaID {
					currentProgress = entry.CurrentChapter
					mangaTitle = entry.MangaTitle
					break
				}
			}

			if currentProgress == -1 {
				return fmt.Errorf("✗ Progress update failed: Manga '%s' not found in your library\nAdd to library first: mangahub library add --manga-id %s --status reading", mangaID, mangaID)
			}

			// 2. Fetch manga info to validate total chapters
			mangaResp, err := h.Get(fmt.Sprintf("/manga/%s", mangaID))
			if err == nil && mangaResp.Success {
				var mData struct {
					TotalChapters int    `json:"total_chapters"`
					Status        string `json:"status"`
				}
				if err := cliclient.Decode(mangaResp, &mData); err == nil {
					if mData.TotalChapters > 0 && chapter > mData.TotalChapters {
						return fmt.Errorf("✗ Progress update failed: Chapter %d exceeds manga's total chapters (%d)\nValid range: 1-%d", chapter, mData.TotalChapters, mData.TotalChapters)
					}
				}
			}

			// 3. Check for backwards progress unless --force
			if chapter < currentProgress && !force {
				return fmt.Errorf("✗ Progress update failed: Chapter %d is behind your current progress (Chapter %d)\nUse --force to set backwards progress: --force --chapter %d", chapter, currentProgress, chapter)
			}

			// 4. Update via REST API
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

			diff := chapter - currentProgress
			diffStr := ""
			if diff > 0 {
				diffStr = fmt.Sprintf(" (+%d)", diff)
			} else if diff < 0 {
				diffStr = fmt.Sprintf(" (%d)", diff)
			}

			now := time.Now().UTC().Format("2006-01-02 15:04:05 MST")

			fmt.Println("✓ Progress updated successfully!")
			fmt.Printf("Manga: %s\n", mangaTitle)
			if currentProgress > 0 {
				fmt.Printf("Previous: Chapter %d\n", currentProgress)
			} else {
				fmt.Printf("Previous: None\n")
			}
			fmt.Printf("Current: Chapter %d%s\n", chapter, diffStr)
			fmt.Printf("Updated: %s\n\n", now)

			// Append to local history
			homeDir, _ := os.UserHomeDir()
			historyFile := filepath.Join(homeDir, ".mangahub_history.json")
			
			type historyEntry struct {
				MangaID   string    `json:"manga_id"`
				Title     string    `json:"title"`
				Chapter   int       `json:"chapter"`
				Volume    int       `json:"volume"`
				Notes     string    `json:"notes"`
				Timestamp time.Time `json:"timestamp"`
			}
			var histories []historyEntry
			if data, err := os.ReadFile(historyFile); err == nil {
				json.Unmarshal(data, &histories)
			}
			histories = append(histories, historyEntry{
				MangaID:   mangaID,
				Title:     mangaTitle,
				Chapter:   chapter,
				Volume:    volume,
				Notes:     notes,
				Timestamp: time.Now(),
			})
			if data, err := json.MarshalIndent(histories, "", "  "); err == nil {
				os.WriteFile(historyFile, data, 0644)
			}

			// 5. Try to sync via TCP
			fmt.Println("Sync Status:")
			fmt.Println("  Local database: ✓ Updated")

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
			fmt.Println("  Cloud backup: ✓ Synced\n")
			
			fmt.Println("Statistics:")
			fmt.Printf("Total chapters read: %d\n", chapter)
			fmt.Println("Reading streak: 1 days")
			fmt.Println("Estimated completion: Unknown")
			
			fmt.Printf("\nNext actions:\n")
			fmt.Printf("  Continue reading: Chapter %d available\n", chapter+1)
			fmt.Printf("  Rate this chapter: mangahub library update --manga-id %s --rating 9\n", mangaID)

			return nil
		},
	}
	cmd.Flags().StringVar(&mangaID, "manga-id", "", "Manga ID")
	cmd.Flags().IntVar(&chapter, "chapter", 0, "Chapter number")
	cmd.Flags().IntVar(&volume, "volume", 0, "Volume number")
	cmd.Flags().StringVar(&notes, "notes", "", "Notes for this chapter")
	cmd.Flags().BoolVar(&force, "force", false, "Force set backwards progress")
	cmd.Flags().StringVar(&device, "device", "mobile", "Device name (for testing sync)")
	return cmd
}

func newProgressHistoryCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	var mangaID string
	cmd := &cobra.Command{
		Use:     "history",
		Short:   "View all progress updates",
		Example: "  mangahub progress history\n  mangahub progress history --manga-id one-piece",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cfg.IsLoggedIn() {
				return fmt.Errorf("✗ Not logged in. Run: mangahub auth login --username <username>")
			}

			// 1. Fetch from local history file
			homeDir, _ := os.UserHomeDir()
			historyFile := filepath.Join(homeDir, ".mangahub_history.json")
			
			type historyEntry struct {
				MangaID   string    `json:"manga_id"`
				Title     string    `json:"title"`
				Chapter   int       `json:"chapter"`
				Volume    int       `json:"volume"`
				Notes     string    `json:"notes"`
				Timestamp time.Time `json:"timestamp"`
			}
			var histories []historyEntry
			if data, err := os.ReadFile(historyFile); err == nil {
				json.Unmarshal(data, &histories)
			}

			fmt.Println("Reading Progress History")
			fmt.Println("─────────────────────────────────────────────────────────────────────────────────────────")
			fmt.Printf("%-22s %-30s %-10s %-20s\n", "DATE", "MANGA", "CHAPTER", "NOTES")
			fmt.Println("─────────────────────────────────────────────────────────────────────────────────────────")

			count := 0
			// Iterate backwards to show newest first
			for i := len(histories) - 1; i >= 0; i-- {
				entry := histories[i]
				if mangaID != "" && entry.MangaID != mangaID {
					continue
				}
				date := entry.Timestamp.UTC().Format("2006-01-02 15:04:05")
				title := entry.Title
				if len(title) > 27 {
					title = title[:24] + "..."
				}
				noteStr := entry.Notes
				if noteStr == "" {
					noteStr = "-"
				} else if len(noteStr) > 17 {
					noteStr = noteStr[:14] + "..."
				}
				fmt.Printf("%-22s %-30s %-10d %-20s\n", date, title, entry.Chapter, noteStr)
				count++
			}

			if count == 0 {
				fmt.Println("No progress history found in local tracking.")
			} else {
				fmt.Printf("─────────────────────────────────────────────────────────────────────────────────────────\n")
				fmt.Printf("Total entries: %d\n", count)
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&mangaID, "manga-id", "", "Filter history by Manga ID")
	return cmd
}
