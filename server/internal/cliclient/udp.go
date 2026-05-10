package cliclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"mangahub/pkg/models"
)

// UDPClient manages a UDP connection to the Notification Server.
type UDPClient struct {
	cfg  *CLIConfig
	conn *net.UDPConn
}

// NewUDPClient creates a UDPClient (not yet connected).
func NewUDPClient(cfg *CLIConfig) *UDPClient {
	return &UDPClient{cfg: cfg}
}

// dial establishes the UDP connection (reused across subscribe/unsubscribe/listen).
func (u *UDPClient) dial() error {
	if u.conn != nil {
		return nil
	}
	serverAddr, err := net.ResolveUDPAddr("udp", u.cfg.UDPAddr())
	if err != nil {
		return fmt.Errorf("invalid UDP address %s: %w", u.cfg.UDPAddr(), err)
	}
	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		return fmt.Errorf("cannot reach UDP notification server at %s — is it running?", u.cfg.UDPAddr())
	}
	u.conn = conn
	return nil
}

// Register sends a register packet to the UDP notification server and
// prints the ACK response.
func (u *UDPClient) Register(userID string) error {
	if err := u.dial(); err != nil {
		return err
	}
	msg := models.RegistrationMsg{Type: "register", UserID: userID}
	// Best Practice — General Error-Handling Pattern:
	// Always check json.Marshal errors instead of discarding with `_`.
	// While marshalling a simple struct rarely fails, ignoring the error
	// masks bugs (e.g. unexportable fields, circular references) and
	// violates the doc's error-handling guidelines.
	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to encode register message: %w", err)
	}
	if _, err := u.conn.Write(b); err != nil {
		return fmt.Errorf("failed to send register packet: %w", err)
	}

	// Read ACK (with 3s deadline per skills-doc best practice)
	u.conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 508)
	n, err := u.conn.Read(buf)
	if err != nil {
		return fmt.Errorf("no ACK received from UDP server (timeout): %w", err)
	}

	var ack models.UDPAck
	if err := json.Unmarshal(buf[:n], &ack); err == nil {
		fmt.Printf("  Server: %s — %s\n", ack.Status, ack.Message)
	}
	return nil
}

// Unregister sends an unregister packet and prints the ACK.
func (u *UDPClient) Unregister(userID string) error {
	if err := u.dial(); err != nil {
		return err
	}
	msg := models.RegistrationMsg{Type: "unregister", UserID: userID}
	// Best Practice — General Error-Handling Pattern:
	// Same as Register — never discard marshal errors.
	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to encode unregister message: %w", err)
	}
	if _, err := u.conn.Write(b); err != nil {
		return fmt.Errorf("failed to send unregister packet: %w", err)
	}

	u.conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 508)
	n, err := u.conn.Read(buf)
	if err != nil {
		return fmt.Errorf("no ACK received (timeout): %w", err)
	}
	var ack models.UDPAck
	if err := json.Unmarshal(buf[:n], &ack); err == nil {
		fmt.Printf("  Server: %s — %s\n", ack.Status, ack.Message)
	}
	return nil
}

// Listen blocks and prints every incoming UDP notification until the context
// is cancelled or the connection is closed.
// Call Register first to start receiving broadcasts.
//
// Best Practice — Mistake #6: "Goroutine Leaks"
// Accept a context so callers can signal shutdown gracefully. A goroutine
// watches ctx.Done() and closes the connection to unblock the blocking Read.
func (u *UDPClient) Listen(ctx context.Context) error {
	if err := u.dial(); err != nil {
		return err
	}

	// Cancel-aware goroutine: close conn when context is done to unblock
	// the blocking u.conn.Read() call below.
	go func() {
		<-ctx.Done()
		u.conn.Close()
	}()

	fmt.Printf("Listening for chapter notifications... (Press Ctrl+C to stop)\n\n")

	// Best Practice — Mistake #4: "Oversized UDP Packets"
	// Buffer capped at 508 bytes — the safe UDP payload size across all networks.
	buf := make([]byte, 508)
	for {
		// No per-read deadline here: we intentionally block waiting for
		// notifications. Shutdown is handled via ctx cancellation above.
		u.conn.SetReadDeadline(time.Time{})
		n, err := u.conn.Read(buf)
		if err != nil {
			// Distinguish context cancellation from real errors.
			if ctx.Err() != nil {
				fmt.Println("\nListener stopped.")
				return nil
			}
			// Likely interrupted by Ctrl-C (conn closed)
			return nil
		}
		var notif models.Notification
		if err := json.Unmarshal(buf[:n], &notif); err == nil {
			ts := time.Unix(notif.Timestamp, 0).Format("15:04:05")
			fmt.Printf("[%s] 🔔 %s — %s\n", ts, notif.MangaID, notif.Message)
		}
	}
}

// Close shuts down the UDP connection.
func (u *UDPClient) Close() {
	if u.conn != nil {
		u.conn.Close()
		u.conn = nil
	}
}
