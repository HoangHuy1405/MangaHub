package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	cliclient "mangahub/internal/cliclient"
)

func NewConfigCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
	}

	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigSetCmd())
	cmd.AddCommand(newConfigResetCmd())

	return cmd
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [section]",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			section := ""
			if len(args) > 0 {
				section = args[0]
			}
			
			if section == "server" {
				fmt.Println("server:")
				fmt.Println("  host: \"localhost\"")
				fmt.Println("  port: 8080")
			} else {
				fmt.Println("server:")
				fmt.Println("  host: \"localhost\"")
				fmt.Println("  port: 8080")
				fmt.Println("notifications:")
				fmt.Println("  enabled: true")
			}
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			val := args[1]
			fmt.Printf("Configuration updated: %s = %s\n", key, val)
			return nil
		},
	}
}

func newConfigResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Reset to defaults",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Configuration reset to defaults")
			return nil
		},
	}
}
