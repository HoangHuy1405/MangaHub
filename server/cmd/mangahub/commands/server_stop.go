package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newServerStopCmd() *cobra.Command {
	var component string

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop MangaHub servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			components := []string{"api-server", "tcp-server", "udp-server", "grpc-server"}
			
			if component != "" {
				if component == "http" {
					components = []string{"api-server"}
				} else if component == "tcp" {
					components = []string{"tcp-server"}
				} else if component == "udp" {
					components = []string{"udp-server"}
				} else if component == "grpc" {
					components = []string{"grpc-server"}
				} else if component == "ws" {
					components = []string{}
					fmt.Println("WebSocket server is managed by the HTTP API server. Stop 'http' instead.")
				} else {
					return fmt.Errorf("unknown component: %s", component)
				}
			}

			stoppedAny := false
			for _, name := range components {
				pid, err := readPID(name)
				if err != nil {
					continue // Not running or PID file missing
				}

				if isProcessRunning(pid) {
					process, err := os.FindProcess(pid)
					if err == nil {
						fmt.Printf("Stopping %s (PID: %d)...\n", name, pid)
						process.Kill()
						stoppedAny = true
					}
				}
				removePID(name)
			}
			
			if !stoppedAny && component == "" {
				fmt.Println("No servers are currently running.")
			} else if component == "" {
				fmt.Println("All servers stopped.")
			}
			return nil
		},
	}
	
	cmd.Flags().StringVar(&component, "component", "", "Stop a specific server component (http, tcp, udp, grpc)")
	return cmd
}
