package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/spf13/cobra"
)

func getUptime(id string) string {
	file := filepath.Join(getRunDir(), id+".pid")
	info, err := os.Stat(file)
	if err != nil {
		return "N/A"
	}
	uptime := time.Since(info.ModTime())
	if uptime < time.Minute {
		return fmt.Sprintf("%ds", int(uptime.Seconds()))
	}
	if uptime < time.Hour {
		return fmt.Sprintf("%dm", int(uptime.Minutes()))
	}
	hours := int(uptime.Hours())
	mins := int(uptime.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, mins)
}

func newServerStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check server status",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("MangaHub Server Status\n")

			table := tablewriter.NewTable(os.Stdout,
				tablewriter.WithConfig(tablewriter.Config{
					Row: tw.CellConfig{
						Formatting:   tw.CellFormatting{AutoWrap: tw.WrapNormal},
						Alignment:    tw.CellAlignment{Global: tw.AlignLeft},
					},
				}),
			)
			table.Header("Service", "Status", "Address", "Uptime", "Load")

			var tableData [][]any

			services := []struct {
				Name     string
				ID       string
				Protocol string
				Address  string
			}{
				{"HTTP API", "api-server", "tcp", "localhost:8080"},
				{"TCP Sync", "tcp-server", "tcp", "localhost:9090"},
				{"UDP Notifications", "udp-server", "udp", "localhost:9091"},
				{"gRPC Internal", "grpc-server", "tcp", "localhost:9092"},
				{"WebSocket Chat", "ws-server", "tcp", "localhost:9093"},
			}

			overallHealth := "✓ Healthy"

			for _, s := range services {
				status := "✗ Error"
				uptime := "-"
				load := "N/A"

				portOpen := isPortOpen(s.Protocol, s.Address)
				
				pidCheck := false
				pid, err := readPID(s.ID)
				if err == nil && isProcessRunning(pid) {
					pidCheck = true
				}
				
				// Handle WS and UDP which are part of API server
				if s.ID == "ws-server" || s.ID == "udp-server" {
					apiPID, apiErr := readPID("api-server")
					if apiErr == nil && isProcessRunning(apiPID) {
						pidCheck = true
					}
				}

				if portOpen || pidCheck {
					status = "✓ Online"
					
					// If it's part of API server, use api-server's uptime
					targetID := s.ID
					if s.ID == "ws-server" || s.ID == "udp-server" {
						targetID = "api-server"
					}
					uptime = getUptime(targetID)
				} else {
					overallHealth = "⚠ Degraded"
				}

				tableData = append(tableData, []any{s.Name, status, s.Address, uptime, load})
			}

			table.Bulk(tableData)
			table.Render()

			fmt.Printf("\nOverall System Health: %s\n", overallHealth)
			
			if overallHealth != "✓ Healthy" {
				fmt.Println("\nIssues Detected:")
				fmt.Println("  Run 'mangahub server health' for detailed diagnostics")
			}

			return nil
		},
	}
	return cmd
}
