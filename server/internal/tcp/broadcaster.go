package tcp

import (
	"encoding/json"
	"fmt"
	"log"

	"mangahub/pkg/models"
)

// broadcastLoop runs in a dedicated goroutine started by Start().
// It drains the broadcast channel and calls broadcastToUser for each update.
// The loop exits cleanly when the quit channel is closed.
func (s *ProgressSyncServer) broadcastLoop() {
	log.Println("[TCP] Broadcast loop started")
	for {
		select {
		case <-s.quit:
			log.Println("[TCP] Broadcast loop stopped")
			return
		case update := <-s.broadcast:
			s.broadcastToUser(update)
		}
	}
}

// broadcastToUser sends the update to every connection registered under
// update.UserID — i.e. all other devices of the same user account.
//
// Concurrency strategy (fix for Bug #1 — delete under RLock):
//
//	Phase 1 — RLock: iterate connections, collect IDs of dead conns.
//	Phase 2 — WLock: delete the dead conns (write operation requires WLock).
//
// This two-phase approach prevents the race condition of calling delete()
// while only holding a read lock.
func (s *ProgressSyncServer) broadcastToUser(update models.ProgressUpdate) {
	payload, err := json.Marshal(update)
	if err != nil {
		log.Printf("[TCP] broadcastToUser marshal error: %v", err)
		return
	}
	line := string(payload) + "\n"

	// ── Phase 1: RLock — read connections, collect failed ones ───────────────
	s.mu.RLock()
	userConns := s.connections[update.UserID]
	var toRemove []string
	sent := 0
	for id, conn := range userConns {
		if _, err := fmt.Fprint(conn, line); err != nil {
			log.Printf("[TCP] write failed to conn %s (user=%s): %v — marking for removal",
				id, update.UserID, err)
			toRemove = append(toRemove, id)
		} else {
			sent++
		}
	}
	s.mu.RUnlock()

	log.Printf("[TCP] broadcast user=%s → sent to %d device(s), %d dead conn(s) removed",
		update.UserID, sent, len(toRemove))

	// ── Phase 2: WLock — remove dead connections (write operation) ───────────
	if len(toRemove) > 0 {
		s.mu.Lock()
		for _, id := range toRemove {
			if c, ok := s.connections[update.UserID][id]; ok {
				c.Close()
				delete(s.connections[update.UserID], id)
			}
		}
		// Clean up empty user entry.
		if len(s.connections[update.UserID]) == 0 {
			delete(s.connections, update.UserID)
		}
		s.mu.Unlock()
	}
}
