package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	cliclient "mangahub/internal/cliclient"
)

func NewProfileCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Profile Management",
	}

	cmd.AddCommand(newProfileCreateCmd())
	cmd.AddCommand(newProfileSwitchCmd())
	cmd.AddCommand(newProfileListCmd())

	return cmd
}

func newProfileCreateCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create new profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("profile name is required")
			}
			fmt.Printf("Profile '%s' created successfully.\n", name)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Profile name")
	return cmd
}

func newProfileSwitchCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "switch",
		Short: "Switch profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("profile name is required")
			}
			fmt.Printf("Switched to profile '%s'.\n", name)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Profile name")
	return cmd
}

func newProfileListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("* default")
			fmt.Println("  work")
			return nil
		},
	}
}
