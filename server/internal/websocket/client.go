package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"

	"mangahub/pkg/models"
)

// Timing constants — per skill (websocket-hub-client) Section 3.
const (
	writeWait  = 10 * time.Second  // Time allowed to write a message.
	pongWait   = 60 * time.Second  // Time allowed to read the next pong.
	pingPeriod = 54 * time.Second  // Must be < pongWait.
	maxMsgSize = 512               // Maximum message size in bytes.
)

// Client is a middleman between the WebSocket connection and the Hub.
// Two goroutines per client: readPump (read from WS) + writePump (write to WS).
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte // buffered — skill mandates make(chan []byte, 256)
	userID   string
	username string
	room     string
}

// readPump reads messages from the WebSocket connection to the Hub.
//
// Skill compliance:
//   - defer unregisters client AND closes connection (checklist #3)
//   - SetPongHandler resets read deadline (checklist #4)
//   - SetReadLimit prevents oversized payloads
func (c *Client) readPump() {
	// ── Cleanup: unregister + close (skill checklist #3) ──────────────────
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMsgSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	// ── Pong handler resets deadline (skill checklist #4) ─────────────────
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, rawMsg, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[WS] read error from user=%s: %v", c.username, err)
			}
			break
		}

		// Parse incoming message to inspect type.
		var incoming models.ChatMessage
		if err := json.Unmarshal(rawMsg, &incoming); err != nil {
			log.Printf("[WS] invalid JSON from user=%s: %v", c.username, err)
			c.sendError("Invalid message format")
			continue
		}

		// Route by message type.
		switch incoming.Type {
		case models.MsgChat:
			// Enrich with server-side metadata and broadcast.
			outgoing := models.ChatMessage{
				Type:      models.MsgChat,
				UserID:    c.userID,
				Username:  c.username,
				Message:   incoming.Message,
				Room:      c.room,
				Timestamp: time.Now().Unix(),
			}
			data, err := json.Marshal(outgoing)
			if err != nil {
				log.Printf("[WS] marshal error: %v", err)
				continue
			}
			c.hub.broadcast <- data

		case models.MsgUsers:
			c.hub.handleUsersRequest(c)

		case models.MsgHistory:
			c.hub.handleHistoryRequest(c)

		case models.MsgPM:
			// Enrich with server-side metadata and broadcast.
			outgoing := models.ChatMessage{
				Type:      models.MsgPM,
				UserID:    c.userID,
				Username:  c.username,
				Target:    incoming.Target,
				Message:   incoming.Message,
				Room:      c.room,
				Timestamp: time.Now().Unix(),
			}
			data, err := json.Marshal(outgoing)
			if err != nil {
				log.Printf("[WS] marshal error: %v", err)
				continue
			}
			c.hub.broadcast <- data

		default:
			c.sendError("Unknown message type: " + string(incoming.Type))
		}
	}
}

// writePump pumps messages from the Hub to the WebSocket connection.
//
// Skill compliance:
//   - Dedicated writePump goroutine prevents concurrent writes (Anti-Pattern #2)
//   - Ping ticker keeps connection alive (Anti-Pattern #3)
//   - CloseMessage sent before close (Anti-Pattern #4)
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel — send clean CloseMessage.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(msg)

			// Drain queued messages into the same write frame for efficiency.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			// ── Ping to keep connection alive (skill checklist #5) ────────
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// sendError sends a JSON error message back to this client only.
func (c *Client) sendError(message string) {
	errMsg := models.ChatMessage{
		Type:      models.MsgError,
		Message:   message,
		Timestamp: time.Now().Unix(),
	}
	data, err := json.Marshal(errMsg)
	if err != nil {
		return
	}
	select {
	case c.send <- data:
	default:
		// Channel full — client is too slow, will be cleaned up.
	}
}
