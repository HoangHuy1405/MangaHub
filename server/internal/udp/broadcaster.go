package udp

import (
	"encoding/json"
	"fmt"
	"log"

	"mangahub/pkg/models"
)

// maxUDPPayload is the safe upper bound for a UDP datagram payload.
// Packets larger than ~508 bytes risk fragmentation or silent drop on
// many network paths. (ref: skills doc §3 Mistake #4)
const maxUDPPayload = 508

// BroadcastNotification sends n to every registered UDP client.
//
// Design notes:
//   - The function is intentionally exported so the REST API chapter hook
//     can call it without any internal coupling.
//   - Errors from individual WriteToUDP calls are logged but do not abort
//     the broadcast — a single unreachable client must not prevent others
//     from receiving the notification.
//   - The clients map is only read here (RLock). Handler goroutines hold WLock
//     for writes, so there is no lock inversion risk.
func (s *NotificationServer) BroadcastNotification(n models.Notification) error {
	payload, err := json.Marshal(n)
	if err != nil {
		return fmt.Errorf("[UDP] broadcast marshal: %w", err)
	}

	if len(payload) > maxUDPPayload {
		return fmt.Errorf("[UDP] payload too large: %d bytes (max %d) — trim the Message field",
			len(payload), maxUDPPayload)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.clients)
	if total == 0 {
		log.Printf("[UDP] broadcast: no registered clients — nothing to send")
		return nil
	}

	failed := 0
	for key, addr := range s.clients {
		if _, err := s.conn.WriteToUDP(payload, addr); err != nil {
			log.Printf("[UDP] broadcast write failed to %s: %v", key, err)
			failed++
			// Continue — do not abort the loop for one bad client.
		}
	}

	log.Printf("[UDP] broadcast manga_id=%s → %d/%d clients notified (%d failed)",
		n.MangaID, total-failed, total, failed)

	return nil
}
