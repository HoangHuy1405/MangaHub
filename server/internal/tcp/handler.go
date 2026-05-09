package tcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"mangahub/pkg/models"
)

// connReadTimeout is the idle deadline for a single TCP connection.
// If the client sends nothing for this duration the scanner will unblock,
// the connection is treated as stale and closed cleanly.
//
// The deadline is reset after every successfully received message, so an
// active client is never evicted.
const connReadTimeout = 5 * time.Minute

// handleConnection runs in its own goroutine (one per accepted connection).
//
// Best practices applied (ref: go-networking-tcp-udp.md):
//   - defer conn.Close()  → prevents fd leaks (Mistake #1)
//   - recover()           → prevents a single panic from crashing the server
//   - SetReadDeadline     → prevents goroutine hang on silent client (Mistake #2)
//   - bufio.Scanner with custom buffer → handles newline-delimited JSON safely
func (s *ProgressSyncServer) handleConnection(connID string, conn net.Conn) {
	// userID is populated on the first valid message received.
	// It is captured in the defer closure for cleanup.
	var userID string

	// ── Cleanup & panic guard ─────────────────────────────────────────────────
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[TCP] panic recovered in conn %s: %v", connID, r)
		}
		conn.Close() // always close — prevents fd leak (Mistake #1)
		if userID != "" {
			s.removeConn(userID, connID)
			log.Printf("[TCP] conn %s (user=%s) cleaned up", connID, userID)
		} else {
			log.Printf("[TCP] conn %s disconnected before sending first message", connID)
		}
	}()

	// ── Initial read deadline ─────────────────────────────────────────────────
	// Prevents goroutine hang when client connects but never sends (Mistake #2).
	if err := conn.SetReadDeadline(time.Now().Add(connReadTimeout)); err != nil {
		log.Printf("[TCP] SetReadDeadline failed on conn %s: %v", connID, err)
		return
	}

	// ── Scanner setup ─────────────────────────────────────────────────────────
	// Default scanner buffer is 64 KB which could truncate large JSON payloads.
	// Increase to 1 MB to be safe.
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 1*1024*1024), 1*1024*1024)

	for scanner.Scan() {
		// ── Reset deadline on each received message ───────────────────────────
		// Keeps long-lived active connections alive past the idle timeout.
		if err := conn.SetReadDeadline(time.Now().Add(connReadTimeout)); err != nil {
			log.Printf("[TCP] SetReadDeadline reset failed on conn %s: %v", connID, err)
			return
		}

		// ── Parse JSON payload ────────────────────────────────────────────────
		var update models.ProgressUpdate
		if err := json.Unmarshal(scanner.Bytes(), &update); err != nil {
			log.Printf("[TCP] invalid JSON from conn %s: %v", connID, err)
			writeJSON(conn, models.TCPResponse{
				Status:    "error",
				Message:   "invalid JSON: " + err.Error(),
				Timestamp: time.Now().Unix(),
			})
			continue
		}

		// ── First message: register this connection under user_id ─────────────
		if userID == "" && update.UserID != "" {
			userID = update.UserID
			s.addConn(userID, connID, conn)
			log.Printf("[TCP] conn %s registered for user=%s", connID, userID)
		}

		log.Printf("[TCP] progress update from user=%s manga=%s chapter=%d",
			update.UserID, update.MangaID, update.Chapter)

		// ── ACK the originating client ────────────────────────────────────────
		writeJSON(conn, models.TCPResponse{
			Status:    "ok",
			Message:   "Progress synced",
			Timestamp: time.Now().Unix(),
		})

		// ── Fan-out to all other devices of this user ─────────────────────────
		// Non-blocking send — if the broadcast channel is full the update is
		// dropped rather than blocking the handler goroutine.
		select {
		case s.broadcast <- update:
		default:
			log.Printf("[TCP] broadcast channel full — dropping update from conn %s", connID)
		}
	}

	// ── Scanner stopped ───────────────────────────────────────────────────────
	if err := scanner.Err(); err != nil {
		// net.Error with Timeout() == true means the read deadline fired.
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			log.Printf("[TCP] conn %s idle timeout — closing", connID)
		} else {
			log.Printf("[TCP] scanner error on conn %s: %v", connID, err)
		}
	}
	// defer handles conn.Close() and removeConn().
}

// writeJSON marshals v to JSON and writes it as a single newline-terminated
// line to conn. Errors are logged but not propagated — a failed write on one
// connection should never abort the handling of others.
func writeJSON(conn net.Conn, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("[TCP] marshal error: %v", err)
		return
	}
	if _, err := fmt.Fprintf(conn, "%s\n", b); err != nil {
		log.Printf("[TCP] write error to %s: %v", conn.RemoteAddr(), err)
	}
}
