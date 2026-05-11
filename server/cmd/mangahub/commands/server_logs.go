package commands

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newServerLogsCmd() *cobra.Command {
	var follow bool
	var level string

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "View server logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			logPath := filepath.Join(getLogDir(), "server.log")
			
			file, err := os.Open(logPath)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("No logs found at", logPath)
					return nil
				}
				return fmt.Errorf("failed to open log file: %w", err)
			}
			defer file.Close()

			reader := bufio.NewReader(file)
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						if follow {
							time.Sleep(500 * time.Millisecond)
							continue
						}
						break
					}
					return err
				}
				
				line = strings.TrimRight(line, "\r\n")
				
				if level != "" {
					if !strings.Contains(strings.ToLower(line), strings.ToLower(level)) {
						continue
					}
				}
				fmt.Println(line)
			}

			return nil
		},
	}
	
	cmd.Flags().BoolVar(&follow, "follow", false, "Follow logs in real-time")
	cmd.Flags().StringVar(&level, "level", "", "Filter logs by level")
	return cmd
}
