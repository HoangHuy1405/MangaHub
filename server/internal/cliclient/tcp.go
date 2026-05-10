package cliclient

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"mangahub/pkg/models"
)

// TCPClient manages a single TCP connection to the Progress Sync Server.
type TCPClient struct {
	cfg  *CLIConfig
	conn net.Conn
}

// NewTCPClient creates a TCPClient (not yet connected).
func NewTCPClient(cfg *CLIConfig) *TCPClient {
	return &TCPClient{cfg: cfg}
}

// Connect dials the TCP sync server and registers this client under userID.
// The first message sent doubles as the registration handshake.
func (t *TCPClient) Connect(userID string) error {
	conn, err := net.DialTimeout("tcp", t.cfg.TCPAddr(), 5*time.Second)
	if err != nil {
		return fmt.Errorf("cannot connect to TCP sync server at %s — is it running?", t.cfg.TCPAddr())
	}
	t.conn = conn

	// Send registration message (first message defines user_id for this conn)
	reg := models.ProgressUpdate{
		UserID:    userID,
		MangaID:   "init",
		Chapter:   0,
		Timestamp: time.Now().Unix(),
	}
	if err := t.sendJSON(reg); err != nil {
		t.conn.Close()
		t.conn = nil
		return fmt.Errorf("registration handshake failed: %w", err)
	}

	// Best Practice — Mistake #2: "Blocking Read Without Timeout"
	// Always set a read deadline before any blocking read to prevent hanging
	// forever if the server never responds.
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		fmt.Printf("  Server ACK: %s\n", scanner.Text())
	}
	// Clear the deadline so future reads on this connection are not affected.
	conn.SetReadDeadline(time.Time{})
	return nil
}

// SendProgress pushes a ProgressUpdate to the TCP server and prints the ACK.
func (t *TCPClient) SendProgress(update models.ProgressUpdate) error {
	if t.conn == nil {
		return fmt.Errorf("not connected — run 'mangahub sync connect' first")
	}
	if err := t.sendJSON(update); err != nil {
		return err
	}

	// Best Practice — Mistake #2: "Blocking Read Without Timeout"
	// Set a deadline before reading the ACK to avoid blocking forever
	// if the server processes the update but never sends a response.
	t.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	scanner := bufio.NewScanner(t.conn)
	if scanner.Scan() {
		var resp models.TCPResponse
		if err := json.Unmarshal(scanner.Bytes(), &resp); err == nil {
			fmt.Printf("  Sync Status: %s — %s\n", resp.Status, resp.Message)
		}
	}
	t.conn.SetReadDeadline(time.Time{}) // clear deadline for future ops
	return nil
}

// Monitor listens for incoming broadcast messages from the TCP server and
// prints each one as it arrives. Blocks until the context is cancelled,
// the connection is closed, or Ctrl-C is received.
//
// Best Practice — Mistake #6: "Goroutine Leaks"
// Accept a context so callers can signal shutdown gracefully instead of
// relying solely on OS signals. The goroutine watches ctx.Done() and
// closes the connection to unblock the scanner.
//
// Best Practice — Mistake #2: "Blocking Read Without Timeout"
// A 60-second rolling deadline detects dead connections that go silent
// without a TCP FIN (e.g. network partition). The deadline is reset
// after every successful read.
func (t *TCPClient) Monitor(ctx context.Context) error {
	if t.conn == nil {
		return fmt.Errorf("not connected — run 'mangahub sync connect' first")
	}

	// Cancel-aware goroutine: close conn when context is done to unblock
	// the blocking scanner.Scan() call below.
	go func() {
		<-ctx.Done()
		t.conn.Close()
	}()

	fmt.Printf("Monitoring real-time sync updates... (Press Ctrl+C to exit)\n\n")

	// Set an initial read deadline to detect dead connections.
	t.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	scanner := bufio.NewScanner(t.conn)
	for scanner.Scan() {
		// Reset deadline after each successful read — connection is alive.
		t.conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		var update models.ProgressUpdate
		if err := json.Unmarshal(scanner.Bytes(), &update); err == nil {
			ts := time.Unix(update.Timestamp, 0).Format("15:04:05")
			fmt.Printf("[%s] ← Device updated: %s → Chapter %d\n",
				ts, update.MangaID, update.Chapter)
		}
	}

	// Distinguish context cancellation from real errors.
	if ctx.Err() != nil {
		fmt.Println("\nMonitor stopped.")
		return nil
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("connection lost: %w", err)
	}
	fmt.Println("\nConnection closed by server.")
	return nil
}

// Close cleanly shuts down the TCP connection.
func (t *TCPClient) Close() {
	if t.conn != nil {
		t.conn.Close()
		t.conn = nil
	}
}

// IsConnected returns true when an active TCP connection exists.
func (t *TCPClient) IsConnected() bool {
	return t.conn != nil
}

func (t *TCPClient) sendJSON(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(t.conn, "%s\n", b)
	return err
}
