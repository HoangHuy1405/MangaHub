package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"mangahub/cmd/mangahub/commands"
	cliclient "mangahub/internal/cliclient"
)

func main() {
	cfg, err := cliclient.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load config: %v\n", err)
		cfg = cliclient.Default()
	}

	root := &cobra.Command{
		Use:   "mangahub",
		Short: "MangaHub CLI — manga tracking with real-time sync",
		Long: `MangaHub CLI provides access to all core features:
  manga discovery, reading progress tracking,
  real-time TCP synchronization, and UDP notifications.`,
		SilenceUsage: true,
	}

	// Wire all commands
	root.AddCommand(
		commands.NewAuthCmd(cfg),
		commands.NewMangaCmd(cfg),
		commands.NewLibraryCmd(cfg),
		commands.NewProgressCmd(cfg),
		commands.NewSyncCmd(cfg),
		commands.NewNotifyCmd(cfg),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
