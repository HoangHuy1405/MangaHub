package commands

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cliclient "mangahub/internal/cliclient"
)

// NewGRPCCmd creates the top-level "grpc" command group for admin/testing.
func NewGRPCCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "grpc",
		Short: "gRPC internal service (admin/testing)",
		Long: `Access the gRPC MangaService directly for administration and testing.
This proves the internal data engine works independently from the REST API.`,
	}
	cmd.AddCommand(
		newGRPCMangaCmd(cfg),
		newGRPCProgressCmd(cfg),
	)
	return cmd
}

// ── manga ─────────────────────────────────────────────────────────────────────

func newGRPCMangaCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "manga",
		Short: "Manga queries via gRPC",
	}
	cmd.AddCommand(
		newGRPCMangaGetCmd(cfg),
		newGRPCMangaSearchCmd(cfg),
	)
	return cmd
}

func newGRPCMangaGetCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	var mangaID string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get manga details by ID via gRPC",
		Example: `  mangahub grpc manga get --id "some-manga-id"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if mangaID == "" {
				return fmt.Errorf("--id is required")
			}

			fmt.Printf("Connecting to gRPC server at %s...\n", cfg.GRPCAddr())

			client := cliclient.NewGRPCClient(cfg)
			if err := client.Connect(); err != nil {
				return fmt.Errorf("✗ %v\n\nTo start the gRPC server:\n  go build -o grpc-server.exe ./cmd/grpc-server && .\\grpc-server.exe", err)
			}
			defer client.Close()

			resp, err := client.GetManga(mangaID)
			if err != nil {
				return handleGRPCError("GetManga", err)
			}

			fmt.Println()
			fmt.Printf("✓ Manga found via gRPC\n")
			fmt.Printf("  ID:             %s\n", resp.GetId())
			fmt.Printf("  Title:          %s\n", resp.GetTitle())
			fmt.Printf("  Author:         %s\n", resp.GetAuthor())
			fmt.Printf("  Genres:         %s\n", resp.GetGenres())
			fmt.Printf("  Status:         %s\n", resp.GetStatus())
			fmt.Printf("  Total Chapters: %d\n", resp.GetTotalChapters())
			fmt.Printf("  Cover URL:      %s\n", resp.GetCoverUrl())
			if desc := resp.GetDescription(); len(desc) > 200 {
				fmt.Printf("  Description:    %s...\n", desc[:200])
			} else if desc != "" {
				fmt.Printf("  Description:    %s\n", desc)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&mangaID, "id", "", "Manga ID to look up (required)")
	return cmd
}

func newGRPCMangaSearchCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	var query, genre, status string
	var page, limit int

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search manga via gRPC",
		Example: `  mangahub grpc manga search --query "one piece"
  mangahub grpc manga search --query "naruto" --genre "Action" --limit 10`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if query == "" && genre == "" {
				return fmt.Errorf("at least --query or --genre is required")
			}

			fmt.Printf("Connecting to gRPC server at %s...\n", cfg.GRPCAddr())

			client := cliclient.NewGRPCClient(cfg)
			if err := client.Connect(); err != nil {
				return fmt.Errorf("✗ %v", err)
			}
			defer client.Close()

			resp, err := client.SearchManga(query, genre, status, page, limit)
			if err != nil {
				return handleGRPCError("SearchManga", err)
			}

			fmt.Printf("\n✓ Search results via gRPC (total: %d)\n\n", resp.GetTotal())

			if len(resp.GetResults()) == 0 {
				fmt.Println("  No manga found matching your query.")
				return nil
			}

			// Print results as a formatted table.
			fmt.Printf("  %-40s %-20s %-10s %s\n", "TITLE", "AUTHOR", "STATUS", "CHAPTERS")
			fmt.Printf("  %-40s %-20s %-10s %s\n", "─────", "──────", "──────", "────────")
			for _, m := range resp.GetResults() {
				title := m.GetTitle()
				if len(title) > 38 {
					title = title[:35] + "..."
				}
				author := m.GetAuthor()
				if len(author) > 18 {
					author = author[:15] + "..."
				}
				fmt.Printf("  %-40s %-20s %-10s %d\n",
					title, author, m.GetStatus(), m.GetTotalChapters())
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Search keyword")
	cmd.Flags().StringVar(&genre, "genre", "", "Filter by genre")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status (ongoing, completed)")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&limit, "limit", 20, "Results per page")
	return cmd
}

// ── progress ──────────────────────────────────────────────────────────────────

func newGRPCProgressCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "progress",
		Short: "Reading progress via gRPC",
	}
	cmd.AddCommand(newGRPCProgressUpdateCmd(cfg))
	return cmd
}

func newGRPCProgressUpdateCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	var mangaID string
	var chapter int

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update reading progress via gRPC",
		Example: `  mangahub grpc progress update --manga-id "some-id" --chapter 42`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cfg.IsLoggedIn() {
				return fmt.Errorf("✗ Not logged in. Run: mangahub auth login --username <username>")
			}
			if mangaID == "" {
				return fmt.Errorf("--manga-id is required")
			}
			if chapter <= 0 {
				return fmt.Errorf("--chapter must be > 0")
			}

			fmt.Printf("Connecting to gRPC server at %s...\n", cfg.GRPCAddr())

			client := cliclient.NewGRPCClient(cfg)
			if err := client.Connect(); err != nil {
				return fmt.Errorf("✗ %v", err)
			}
			defer client.Close()

			// Use a string user ID — the gRPC server parses it internally.
			userID := cfg.Username
			// If we can extract numeric ID from the token, prefer that.
			// For now, use username as user_id since auth system stores numeric IDs.
			// The CLI auth flow stores the numeric user_id after login in some cases.
			// Fallback: just use "1" as a placeholder if username is the only thing we have.
			// In a real setup, the JWT claims would provide this.
			if uid := extractUserIDFromConfig(cfg); uid != "" {
				userID = uid
			}

			resp, err := client.UpdateProgress(userID, mangaID, chapter)
			if err != nil {
				return handleGRPCError("UpdateProgress", err)
			}

			if resp.GetSuccess() {
				fmt.Printf("\n✓ %s\n", resp.GetMessage())
			} else {
				fmt.Printf("\n✗ %s\n", resp.GetMessage())
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&mangaID, "manga-id", "", "Manga ID (required)")
	cmd.Flags().IntVar(&chapter, "chapter", 0, "Current chapter number (required)")
	return cmd
}

// extractUserIDFromConfig tries to get a numeric user ID.
// In MangaHub, the CLI stores the JWT token but not the numeric user_id separately.
// For gRPC testing, we accept the username and let the server handle it.
func extractUserIDFromConfig(cfg *cliclient.CLIConfig) string {
	// If username looks numeric (from some auth flows), use it directly.
	if _, err := strconv.Atoi(cfg.Username); err == nil {
		return cfg.Username
	}
	// Otherwise return empty — the command will use username as user_id.
	return ""
}

// handleGRPCError formats a gRPC error for CLI display using status codes.
func handleGRPCError(rpc string, err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return fmt.Errorf("✗ %s failed: %v", rpc, err)
	}

	switch st.Code() {
	case codes.NotFound:
		return fmt.Errorf("✗ Not found: %s", st.Message())
	case codes.InvalidArgument:
		return fmt.Errorf("✗ Invalid input: %s", st.Message())
	case codes.Unavailable:
		return fmt.Errorf("✗ gRPC server unavailable. Is grpc-server running?\n\nTo start:\n  go build -o grpc-server.exe ./cmd/grpc-server && .\\grpc-server.exe")
	default:
		return fmt.Errorf("✗ %s failed [%s]: %s", rpc, st.Code(), st.Message())
	}
}
