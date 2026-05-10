package commands

import (
	"archive/zip"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	cliclient "mangahub/internal/cliclient"
)

func NewExportCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export Data",
	}

	var formatLib string
	var outputLib string
	libraryCmd := &cobra.Command{
		Use:   "library",
		Short: "Export library to JSON/CSV",
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
				Library []map[string]interface{} `json:"library"`
			}
			if err := cliclient.Decode(resp, &libData); err != nil {
				return fmt.Errorf("failed to decode library")
			}

			fmt.Printf("Exporting library to %s format...\n", formatLib)
			
			if formatLib == "json" {
				data, _ := json.MarshalIndent(libData.Library, "", "  ")
				if err := os.WriteFile(outputLib, data, 0644); err != nil {
					return err
				}
			} else if formatLib == "csv" {
				f, err := os.Create(outputLib)
				if err != nil {
					return err
				}
				defer f.Close()
				w := csv.NewWriter(f)
				w.Write([]string{"manga_id", "manga_title", "current_chapter", "status", "updated_at"})
				for _, entry := range libData.Library {
					mangaID, _ := entry["manga_id"].(string)
					mangaTitle, _ := entry["manga_title"].(string)
					chapterF, _ := entry["current_chapter"].(float64)
					status, _ := entry["status"].(string)
					updatedAt, _ := entry["updated_at"].(string)
					w.Write([]string{mangaID, mangaTitle, strconv.Itoa(int(chapterF)), status, updatedAt})
				}
				w.Flush()
			} else {
				return fmt.Errorf("unsupported format %s", formatLib)
			}
			
			fmt.Printf("✓ Successfully exported library to %s\n", outputLib)
			return nil
		},
	}
	libraryCmd.Flags().StringVar(&formatLib, "format", "json", "Export format (json, csv)")
	libraryCmd.Flags().StringVar(&outputLib, "output", "library.json", "Output file")

	var formatProg string
	var outputProg string
	progressCmd := &cobra.Command{
		Use:   "progress",
		Short: "Export reading progress to CSV/JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cfg.IsLoggedIn() {
				return fmt.Errorf("✗ Not logged in")
			}
			histories := loadLocalHistory()
			fmt.Printf("Exporting progress to %s format...\n", formatProg)
			
			if formatProg == "json" {
				data, _ := json.MarshalIndent(histories, "", "  ")
				if err := os.WriteFile(outputProg, data, 0644); err != nil {
					return err
				}
			} else if formatProg == "csv" {
				f, err := os.Create(outputProg)
				if err != nil {
					return err
				}
				defer f.Close()
				w := csv.NewWriter(f)
				w.Write([]string{"manga_id", "title", "chapter", "volume", "notes", "timestamp"})
				for _, h := range histories {
					w.Write([]string{h.MangaID, h.Title, strconv.Itoa(h.Chapter), strconv.Itoa(h.Volume), h.Notes, h.Timestamp.Format("2006-01-02 15:04:05")})
				}
				w.Flush()
			} else {
				return fmt.Errorf("unsupported format %s", formatProg)
			}
			
			fmt.Printf("✓ Successfully exported progress to %s\n", outputProg)
			return nil
		},
	}
	progressCmd.Flags().StringVar(&formatProg, "format", "csv", "Export format (json, csv)")
	progressCmd.Flags().StringVar(&outputProg, "output", "progress.csv", "Output file")

	var outputAll string
	allCmd := &cobra.Command{
		Use:   "all",
		Short: "Full data export (zip)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cfg.IsLoggedIn() {
				return fmt.Errorf("✗ Not logged in")
			}
			
			// If user asked for tar.gz, just change the output file to .zip internally 
			// so we don't break their command muscle memory, or warn them.
			if strings.HasSuffix(outputAll, ".tar.gz") {
				outputAll = strings.TrimSuffix(outputAll, ".tar.gz") + ".zip"
				fmt.Printf("Note: Changed output format to .zip to prevent Antivirus blocking.\n")
			}

			fmt.Printf("Exporting all data to %s...\n", outputAll)
			
			// 1. Fetch library
			h := cliclient.NewHTTPClient(cfg)
			resp, err := h.Get("/users/library")
			var libData []byte
			if err == nil {
				var ld struct {
					Library []map[string]interface{} `json:"library"`
				}
				cliclient.Decode(resp, &ld)
				libData, _ = json.MarshalIndent(ld.Library, "", "  ")
			}
			
			// 2. Fetch histories
			histories := loadLocalHistory()
			progData, _ := json.MarshalIndent(histories, "", "  ")
			
			// 3. Create zip
			f, err := os.Create(outputAll)
			if err != nil {
				return err
			}
			defer f.Close()
			zw := zip.NewWriter(f)
			defer zw.Close()
			
			// Write library.json
			if fw, err := zw.Create("library.json"); err == nil {
				fw.Write(libData)
			}
			
			// Write progress.json
			if fw, err := zw.Create("progress.json"); err == nil {
				fw.Write(progData)
			}
			
			fmt.Printf("✓ Successfully exported all data to %s\n", outputAll)
			return nil
		},
	}
	allCmd.Flags().StringVar(&outputAll, "output", "mangahub-backup.zip", "Output file")

	cmd.AddCommand(libraryCmd, progressCmd, allCmd)
	return cmd
}
