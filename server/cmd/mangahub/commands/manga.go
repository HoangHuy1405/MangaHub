package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	cliclient "mangahub/internal/cliclient"
)

func NewMangaCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "manga",
		Short: "Manga management (search, info, list)",
	}
	cmd.AddCommand(
		newMangaSearchCmd(cfg),
		newMangaInfoCmd(cfg),
		newMangaListCmd(cfg),
	)
	return cmd
}

// ── search ────────────────────────────────────────────────────────────────────

func newMangaSearchCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	var genre, status string
	var limit int
	cmd := &cobra.Command{
		Use:     "search <query>",
		Short:   "Search for manga by title, genre, or status",
		Example: "  mangahub manga search \"attack on titan\"\n  mangahub manga search \"romance\" --genre romance --status completed",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			fmt.Printf("Searching for \"%s\"...\n", query)

			path := fmt.Sprintf("/manga?search=%s", query)
			if genre != "" {
				path += "&genre=" + genre
			}
			if status != "" {
				path += "&status=" + status
			}
			if limit > 0 {
				path += fmt.Sprintf("&pageSize=%d", limit)
			}

			h := cliclient.NewHTTPClient(cfg)
			resp, err := h.Get(path)
			if err != nil {
				return err
			}
			if !resp.Success {
				return fmt.Errorf("search failed: %s", resp.Message)
			}

			var data struct {
				Manga []struct {
					ID            string `json:"id"`
					Title         string `json:"title"`
					Author        string `json:"author"`
					Status        string `json:"status"`
					TotalChapters int    `json:"total_chapters"`
				} `json:"manga"`
				Meta struct {
					Total int `json:"total"`
				} `json:"meta"`
			}
			if err := json.Unmarshal(resp.Data, &data); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			if len(data.Manga) == 0 {
				fmt.Printf("No manga found matching \"%s\".\n", query)
				fmt.Println("Suggestions:")
				fmt.Println("  - Check spelling and try again")
				fmt.Println("  - Use broader search terms")
				return nil
			}

			fmt.Printf("Found %d result(s):\n\n", data.Meta.Total)
			fmt.Printf("%-28s %-35s %-20s %-12s %s\n", "ID", "Title", "Author", "Status", "Chapters")
			fmt.Println(strings.Repeat("─", 105))
			for _, m := range data.Manga {
				title := m.Title
				if len(title) > 33 {
					title = title[:30] + "..."
				}
				author := m.Author
				if len(author) > 18 {
					author = author[:15] + "..."
				}
				fmt.Printf("%-28s %-35s %-20s %-12s %d\n",
					m.ID, title, author, m.Status, m.TotalChapters)
			}
			fmt.Println()
			fmt.Println("Use 'mangahub manga info <id>' to view details")
			fmt.Println("Use 'mangahub library add --manga-id <id>' to add to your library")
			return nil
		},
	}
	cmd.Flags().StringVar(&genre, "genre", "", "Filter by genre")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status (ongoing/completed)")
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum results to return")
	return cmd
}

// ── info ──────────────────────────────────────────────────────────────────────

func newMangaInfoCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	return &cobra.Command{
		Use:     "info <manga-id>",
		Short:   "View detailed information about a manga",
		Example: "  mangahub manga info one-piece",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			h := cliclient.NewHTTPClient(cfg)
			resp, err := h.Get("/manga/" + id)
			if err != nil {
				return err
			}
			if !resp.Success {
				return fmt.Errorf("✗ Manga not found: '%s'\n  Try searching instead:\n  mangahub manga search \"%s\"", id, id)
			}

			var m struct {
				ID            string `json:"id"`
				Title         string `json:"title"`
				Author        string `json:"author"`
				Genres        string `json:"genres"`
				Status        string `json:"status"`
				TotalChapters int    `json:"total_chapters"`
				Description   string `json:"description"`
				Year          int    `json:"year"`
			}
			if err := json.Unmarshal(resp.Data, &m); err != nil {
				return fmt.Errorf("failed to parse manga data: %w", err)
			}

			border := strings.Repeat("─", 69)
			fmt.Printf("┌%s┐\n", border)
			title := strings.ToUpper(m.Title)
			pad := (69 - len(title)) / 2
			if pad < 0 {
				pad = 0
			}
			fmt.Printf("│%s%s%s│\n", strings.Repeat(" ", pad), title, strings.Repeat(" ", 69-pad-len(title)))
			fmt.Printf("└%s┘\n\n", border)
			fmt.Printf("ID:             %s\n", m.ID)
			fmt.Printf("Title:          %s\n", m.Title)
			fmt.Printf("Author:         %s\n", m.Author)
			fmt.Printf("Genres:         %s\n", m.Genres)
			fmt.Printf("Status:         %s\n", m.Status)
			fmt.Printf("Year:           %d\n", m.Year)
			fmt.Printf("Total Chapters: %d\n\n", m.TotalChapters)
			if m.Description != "" {
				desc := m.Description
				if len(desc) > 200 {
					desc = desc[:200] + "..."
				}
				fmt.Printf("Description:\n%s\n\n", desc)
			}
			fmt.Println("Actions:")
			fmt.Printf("  Add to Library: mangahub library add --manga-id %s --status reading\n", id)
			fmt.Printf("  Update Progress: mangahub progress update --manga-id %s --chapter <number>\n", id)
			return nil
		},
	}
}

// ── list ──────────────────────────────────────────────────────────────────────

func newMangaListCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	var genre string
	var page, limit int
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all manga in the database",
		Example: "  mangahub manga list\n  mangahub manga list --genre shounen --page 2",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := fmt.Sprintf("/manga?page=%d&pageSize=%d", page, limit)
			if genre != "" {
				path += "&genre=" + genre
			}
			h := cliclient.NewHTTPClient(cfg)
			resp, err := h.Get(path)
			if err != nil {
				return err
			}
			if !resp.Success {
				return fmt.Errorf("failed to list manga: %s", resp.Message)
			}
			var data struct {
				Manga []struct {
					ID     string `json:"id"`
					Title  string `json:"title"`
					Status string `json:"status"`
					Year   int    `json:"year"`
				} `json:"manga"`
				Meta struct {
					Page     int `json:"page"`
					Pages    int `json:"pages"`
					Total    int `json:"total"`
					PageSize int `json:"pageSize"`
				} `json:"meta"`
			}
			if err := json.Unmarshal(resp.Data, &data); err != nil {
				return fmt.Errorf("parse error: %w", err)
			}
			fmt.Printf("Manga List — Page %d/%d (Total: %d)\n\n",
				data.Meta.Page, data.Meta.Pages, data.Meta.Total)
			fmt.Printf("%-30s %-40s %-12s %s\n", "ID", "Title", "Status", "Year")
			fmt.Println(strings.Repeat("─", 90))
			for _, m := range data.Manga {
				title := m.Title
				if len(title) > 38 {
					title = title[:35] + "..."
				}
				fmt.Printf("%-30s %-40s %-12s %d\n", m.ID, title, m.Status, m.Year)
			}
			if data.Meta.Page < data.Meta.Pages {
				fmt.Printf("\nNext page: mangahub manga list --page %d\n", data.Meta.Page+1)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&genre, "genre", "", "Filter by genre")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&limit, "limit", 20, "Results per page")
	return cmd
}
