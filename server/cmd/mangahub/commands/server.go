package commands

import (
	"os"

	"github.com/spf13/cobra"

	cliclient "mangahub/internal/cliclient"
)

func NewServerCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Manage MangaHub servers",
	}
	
	cmd.AddCommand(newServerStartCmd())
	cmd.AddCommand(newServerStatusCmd())
	cmd.AddCommand(newServerStopCmd())
	cmd.AddCommand(newServerHealthCmd())
	cmd.AddCommand(newServerLogsCmd())
	
	return cmd
}

func getServerDir() string {
	if _, err := os.Stat("go.mod"); err == nil {
		return "."
	}
	if _, err := os.Stat("server/go.mod"); err == nil {
		return "server"
	}
	return "."
}

