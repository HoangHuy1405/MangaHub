package commands

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/spf13/cobra"

	cliclient "mangahub/internal/cliclient"
)

func NewServerCmd(cfg *cliclient.CLIConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Manage MangaHub servers",
	}
	cmd.AddCommand(newServerStartCmd())
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

func newServerStartCmd() *cobra.Command {
	var httpOnly, udpOnly, tcpOnly, grpcOnly, all bool

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start MangaHub servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			startAPI := httpOnly || all
			startTCP := tcpOnly || all
			startUDP := udpOnly || all
			startGRPC := grpcOnly || all

			if !startAPI && !startTCP && !startUDP && !startGRPC {
				startAPI = true // Default to API server
			}

			serverDir := getServerDir()
			var cmds []*exec.Cmd

			startServer := func(name, path string) error {
				var execCmd *exec.Cmd
				binName := name
				if runtime.GOOS == "windows" {
					binName += ".exe"
				}

				if _, err := os.Stat(filepath.Join(serverDir, binName)); err == nil {
					execCmd = exec.Command(filepath.Join(".", binName))
				} else {
					execCmd = exec.Command("go", "run", path)
				}
				
				execCmd.Dir = serverDir
				execCmd.Stdout = os.Stdout
				execCmd.Stderr = os.Stderr

				if err := execCmd.Start(); err != nil {
					return fmt.Errorf("failed to start %s: %w", name, err)
				}
				cmds = append(cmds, execCmd)
				return nil
			}

			if startAPI {
				fmt.Println("Starting API Server...")
				if err := startServer("api-server", "./cmd/api-server"); err != nil {
					return err
				}
			}
			if startTCP {
				fmt.Println("Starting TCP Server...")
				if err := startServer("tcp-server", "./cmd/tcp-server"); err != nil {
					return err
				}
			}
			if startUDP {
				fmt.Println("Starting UDP Server...")
				if err := startServer("udp-server", "./cmd/udp-server"); err != nil {
					return err
				}
			}
			if startGRPC {
				fmt.Println("Starting gRPC Server...")
				if err := startServer("grpc-server", "./cmd/grpc-server"); err != nil {
					return err
				}
			}

			// Handle graceful shutdown
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM)

			errCh := make(chan error, len(cmds))
			for _, cmdObj := range cmds {
				go func(c *exec.Cmd) {
					errCh <- c.Wait()
				}(cmdObj)
			}

			select {
			case <-c:
				fmt.Println("\nStopping servers...")
				for _, cmdObj := range cmds {
					if err := cmdObj.Process.Signal(os.Interrupt); err != nil {
						cmdObj.Process.Kill()
					}
				}
			case err := <-errCh:
				if err != nil {
					if exitErr, ok := err.(*exec.ExitError); ok {
						if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.Signaled() {
							return nil
						}
					}
					return fmt.Errorf("server exited with error: %w", err)
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&httpOnly, "http-only", false, "Start only the HTTP API server")
	cmd.Flags().BoolVar(&udpOnly, "udp-only", false, "Start only the UDP server")
	cmd.Flags().BoolVar(&tcpOnly, "tcp-only", false, "Start only the TCP server")
	cmd.Flags().BoolVar(&grpcOnly, "grpc-only", false, "Start only the gRPC server")
	cmd.Flags().BoolVar(&all, "all", false, "Start all servers")

	return cmd
}
