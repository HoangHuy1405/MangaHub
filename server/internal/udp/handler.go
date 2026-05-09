package udp

import (
	"encoding/json"
	"log"
	"net"
	"time"

	"mangahub/pkg/models"
)

// handlePacket processes a single UDP datagram. It runs in its own goroutine
// so that the main read loop in Start() is never blocked.
//
// Best practices applied:
//   - recover() at the top — a panic in one packet handler must not crash the server.
//   - All map writes are performed under WLock.
func (s *NotificationServer) handlePacket(data []byte, addr *net.UDPAddr) {
	// ── Panic guard ───────────────────────────────────────────────────────────
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[UDP] panic recovered while handling packet from %s: %v", addr, r)
		}
	}()

	// ── Parse registration message ────────────────────────────────────────────
	var msg models.RegistrationMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("[UDP] invalid packet from %s: %v", addr, err)
		s.sendACK(addr, "error", "invalid JSON: "+err.Error())
		return
	}

	key := addr.String() // "ip:port" — unique client identifier

	switch msg.Type {
	case "register":
		s.mu.Lock()
		s.clients[key] = addr
		s.mu.Unlock()
		log.Printf("[UDP] registered client %s (user_id=%s)", key, msg.UserID)
		s.sendACK(addr, "registered", "You will receive chapter-release notifications")

	case "unregister":
		s.mu.Lock()
		delete(s.clients, key)
		s.mu.Unlock()
		log.Printf("[UDP] unregistered client %s", key)
		s.sendACK(addr, "unregistered", "Notifications stopped")

	default:
		log.Printf("[UDP] unknown packet type %q from %s", msg.Type, addr)
		s.sendACK(addr, "error", "unknown type: "+msg.Type)
	}
}

// sendACK marshals a UDPAck and writes it back to the originating client.
// Errors are logged — a failed ACK must not panic or stop the server.
func (s *NotificationServer) sendACK(addr *net.UDPAddr, status, message string) {
	ack := models.UDPAck{
		Status:    status,
		Message:   message,
		Timestamp: time.Now().Unix(),
	}
	b, err := json.Marshal(ack)
	if err != nil {
		log.Printf("[UDP] sendACK marshal error: %v", err)
		return
	}
	if _, err := s.conn.WriteToUDP(b, addr); err != nil {
		log.Printf("[UDP] sendACK write error to %s: %v", addr, err)
	}
}
