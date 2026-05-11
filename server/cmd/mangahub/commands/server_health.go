package commands

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	_ "github.com/glebarez/go-sqlite"

	"mangahub/pkg/utils/config"
)

func newServerHealthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Detailed health check",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig()
			dbPath := "data/mangahub.db"
			if err == nil && cfg.API_CONFIG.DB_PATH != "" {
				dbPath = cfg.API_CONFIG.DB_PATH
			}

			fmt.Println("Database:")
			
			dbInfo, err := os.Stat(dbPath)
			if err != nil {
				fmt.Println("Connection: ✗ Offline")
				fmt.Printf("Error: %v\n", err)
			} else {
				db, err := sql.Open("sqlite", dbPath)
				if err == nil {
					defer db.Close()
					if err := db.Ping(); err == nil {
						fmt.Println("Connection: ✓ Active")
						
						var tableCount int
						err = db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%';").Scan(&tableCount)
						if err == nil {
							fmt.Printf("Size: %.2f MB\n", float64(dbInfo.Size())/1024/1024)
							fmt.Printf("Tables: %d\n", tableCount)
						}
					} else {
						fmt.Println("Connection: ✗ Error")
					}
				} else {
					fmt.Println("Connection: ✗ Error")
				}
			}
			
			// We can get backup info by checking data folder
			if backupInfo, err := os.Stat("data/mangahub-backup.zip"); err == nil {
				fmt.Printf("  Last backup: %s\n\n", backupInfo.ModTime().Format("2006-01-02 15:04:05"))
			} else {
				fmt.Println("  Last backup: No backup found\n")
			}
			
			// Check if API server is answering on its port
			if isPortOpen("tcp", "localhost:8080") {
				fmt.Println("System Metrics: Active (Real-time monitoring requires gopsutil or external metric server)")
			} else {
				fmt.Println("System Metrics: Unavailable (API server not running)")
			}
			
			return nil
		},
	}
	return cmd
}
