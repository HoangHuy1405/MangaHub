package commands

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func getMangaHubDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".mangahub"
	}
	return filepath.Join(home, ".mangahub")
}

func getRunDir() string {
	return filepath.Join(getMangaHubDir(), "run")
}

func getLogDir() string {
	return filepath.Join(getMangaHubDir(), "logs")
}

func savePID(name string, pid int) error {
	dir := getRunDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	file := filepath.Join(dir, name+".pid")
	return os.WriteFile(file, []byte(strconv.Itoa(pid)), 0644)
}

func readPID(name string) (int, error) {
	file := filepath.Join(getRunDir(), name+".pid")
	data, err := os.ReadFile(file)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func removePID(name string) error {
	file := filepath.Join(getRunDir(), name+".pid")
	return os.Remove(file)
}

func isProcessRunning(pid int) bool {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/NH")
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			return false
		}
		return strings.Contains(out.String(), strconv.Itoa(pid))
	}
	
	// Fallback for Unix
	_, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
	return err == nil
}

func isPortOpen(protocol, address string) bool {
	if protocol == "udp" {
		// net.DialTimeout always succeeds for UDP even if nothing is listening.
		// We cannot reliably check if a UDP port is open this way.
		// Rely entirely on PID checks instead.
		return false
	}
	
	conn, err := net.DialTimeout(protocol, address, 1*time.Second)
	if err != nil {
		return false
	}
	if conn != nil {
		conn.Close()
		return true
	}
	return false
}
