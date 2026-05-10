package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	cliclient "mangahub/internal/cliclient"
)

func NewStatsCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	var fromDate string
	var toDate string

	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Statistics and Analytics",
		RunE: func(cmd *cobra.Command, args []string) error {
			if fromDate != "" || toDate != "" {
				if !cfg.IsLoggedIn() {
					return fmt.Errorf("✗ Not logged in")
				}
				
				// Parse dates
				var fromTime, toTime time.Time
				var err error
				if fromDate != "" {
					fromTime, err = time.Parse("2006-01-02", fromDate)
					if err != nil {
						return fmt.Errorf("invalid from date format, use YYYY-MM-DD")
					}
				}
				if toDate != "" {
					toTime, err = time.Parse("2006-01-02", toDate)
					if err != nil {
						return fmt.Errorf("invalid to date format, use YYYY-MM-DD")
					}
					// Include the whole day
					toTime = toTime.Add(24*time.Hour - time.Second)
				} else {
					toTime = time.Now()
				}
				
				histories := loadLocalHistory()
				
				chaptersRead := 0
				mangaInteracted := make(map[string]bool)
				
				for _, h := range histories {
					if (!fromTime.IsZero() && h.Timestamp.Before(fromTime)) || h.Timestamp.After(toTime) {
						continue
					}
					chaptersRead++
					mangaInteracted[h.MangaID] = true
				}
				
				// Estimate reading time: 5 mins per chapter
				readingTimeHours := float64(chaptersRead*5) / 60.0

				fmt.Printf("Statistics from %s to %s\n", fromDate, toDate)
				fmt.Println("─────────────────────────────────────────────────────────────")
				fmt.Printf("Chapters Read: %d\n", chaptersRead)
				fmt.Printf("Manga Interacted: %d\n", len(mangaInteracted))
				fmt.Printf("Estimated Reading Time: %.1f hours\n", readingTimeHours)
				return nil
			}
			return cmd.Help()
		},
	}
	cmd.Flags().StringVar(&fromDate, "from", "", "Start date (e.g. 2024-01-01)")
	cmd.Flags().StringVar(&toDate, "to", "", "End date (e.g. 2024-12-31)")

	overviewCmd := &cobra.Command{
		Use:   "overview",
		Short: "View personal reading statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cfg.IsLoggedIn() {
				return fmt.Errorf("✗ Not logged in")
			}
			h := cliclient.NewHTTPClient(cfg)
			resp, err := h.Get("/users/library")
			if err != nil {
				return fmt.Errorf("failed to get library: %w", err)
			}
			
			var libData struct {
				Library []struct {
					MangaID        string `json:"manga_id"`
					CurrentChapter int    `json:"current_chapter"`
					Status         string `json:"status"`
				} `json:"library"`
			}
			if err := cliclient.Decode(resp, &libData); err != nil {
				return fmt.Errorf("failed to decode library")
			}
			
			totalChapters := 0
			completedManga := 0
			for _, entry := range libData.Library {
				totalChapters += entry.CurrentChapter
				if entry.Status == "completed" {
					completedManga++
				}
			}

			// Calculate streak
			streak := calculateStreak()

			fmt.Println("Reading Statistics Overview")
			fmt.Println("─────────────────────────────────────────────────────────────")
			fmt.Printf("Total Manga in Library: %d\n", len(libData.Library))
			fmt.Printf("Total Chapters Read: %d\n", totalChapters)
			fmt.Printf("Manga Completed: %d\n", completedManga)
			fmt.Printf("Current Streak: %d days\n", streak)
			return nil
		},
	}

	detailedCmd := &cobra.Command{
		Use:   "detailed",
		Short: "Detailed breakdown",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cfg.IsLoggedIn() {
				return fmt.Errorf("✗ Not logged in")
			}
			h := cliclient.NewHTTPClient(cfg)
			resp, err := h.Get("/users/library")
			if err != nil {
				return fmt.Errorf("failed to get library: %w", err)
			}
			
			var libData struct {
				Library []struct {
					MangaID        string `json:"manga_id"`
					MangaGenres    string `json:"manga_genres"`
				} `json:"library"`
			}
			if err := cliclient.Decode(resp, &libData); err != nil {
				return fmt.Errorf("failed to decode library")
			}
			
			genreCounts := make(map[string]int)
			totalGenres := 0
			for _, entry := range libData.Library {
				genres := strings.Split(entry.MangaGenres, ",")
				for _, g := range genres {
					g = strings.TrimSpace(g)
					if g != "" {
						genreCounts[g]++
						totalGenres++
					}
				}
			}
			
			// Sort genres by count
			type genreStat struct {
				name  string
				count int
			}
			var genreList []genreStat
			for k, v := range genreCounts {
				genreList = append(genreList, genreStat{k, v})
			}
			sort.Slice(genreList, func(i, j int) bool {
				return genreList[i].count > genreList[j].count
			})

			histories := loadLocalHistory()
			dayCounts := make(map[string]int)
			for _, h := range histories {
				day := h.Timestamp.Weekday().String()
				dayCounts[day]++
			}

			fmt.Println("Detailed Reading Breakdown")
			fmt.Println("─────────────────────────────────────────────────────────────")
			fmt.Println("By Genre:")
			if len(genreList) == 0 {
				fmt.Println("  No genre data available")
			} else {
				for i, g := range genreList {
					if i >= 5 { // Show top 5
						break
					}
					pct := float64(g.count) / float64(totalGenres) * 100
					fmt.Printf("  %s: %.1f%% (%d)\n", g.name, pct, g.count)
				}
			}
			
			fmt.Println("\nActivity by Day:")
			days := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
			hasActivity := false
			for _, d := range days {
				if dayCounts[d] > 0 {
					fmt.Printf("  %s: %d chapters\n", d, dayCounts[d])
					hasActivity = true
				}
			}
			if !hasActivity {
				fmt.Println("  No activity recorded yet.")
			}
			return nil
		},
	}

	cmd.AddCommand(overviewCmd, detailedCmd)
	return cmd
}

// Since progress.go uses historyEntry as an anonymous struct we will define it here locally too, 
// to avoid conflicts or re-definition issues across the package, or just keep it unexported.
// In Go, multiple files in the same package can't redefine the same type, so we need to be careful.
// Wait, in progress.go, `historyEntry` is defined inside the function `newProgressUpdateCmd` and `newProgressHistoryCmd`.
// So defining it at the package level here is safe from conflicts.

type historyEntry struct {
	MangaID   string    `json:"manga_id"`
	Title     string    `json:"title"`
	Chapter   int       `json:"chapter"`
	Volume    int       `json:"volume"`
	Notes     string    `json:"notes"`
	Timestamp time.Time `json:"timestamp"`
}

func loadLocalHistory() []historyEntry {
	homeDir, _ := os.UserHomeDir()
	historyFile := filepath.Join(homeDir, ".mangahub_history.json")
	var histories []historyEntry
	if data, err := os.ReadFile(historyFile); err == nil {
		json.Unmarshal(data, &histories)
	}
	return histories
}

func calculateStreak() int {
	histories := loadLocalHistory()
	if len(histories) == 0 {
		return 0
	}
	
	// Sort descending by timestamp
	sort.Slice(histories, func(i, j int) bool {
		return histories[i].Timestamp.After(histories[j].Timestamp)
	})
	
	streak := 0
	lastDate := time.Now().Truncate(24 * time.Hour)
	
	// Check if today or yesterday has activity to start streak
	hasRecentActivity := false
	for _, h := range histories {
		hDate := h.Timestamp.Truncate(24 * time.Hour)
		if hDate.Equal(lastDate) || hDate.Equal(lastDate.Add(-24*time.Hour)) {
			hasRecentActivity = true
			break
		}
	}
	if !hasRecentActivity {
		return 0
	}

	activeDays := make(map[string]bool)
	for _, h := range histories {
		activeDays[h.Timestamp.Format("2006-01-02")] = true
	}

	// Count backwards from today
	currDate := time.Now()
	// If no activity today, start from yesterday
	if !activeDays[currDate.Format("2006-01-02")] {
		currDate = currDate.AddDate(0, 0, -1)
	}

	for {
		dateStr := currDate.Format("2006-01-02")
		if activeDays[dateStr] {
			streak++
			currDate = currDate.AddDate(0, 0, -1)
		} else {
			break
		}
	}
	
	return streak
}
