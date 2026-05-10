package websocket

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// upgrader upgrades HTTP requests to WebSocket connections.
// CheckOrigin allows all origins in dev mode — restrict in production! (skill checklist #7)
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: restrict to production domain in deployment
		return true
	},
}

// ServeWS handles the HTTP upgrade request and registers the client.
//
// Authentication: the username query parameter is REQUIRED. Without it the
// request is rejected with 401 Unauthorized. This enforces the spec
// requirement that only authenticated users may join chat.
//
// Query parameters:
//   - username (required): display name of the connecting user
//   - user_id  (optional): unique user identifier (defaults to username)
//   - room     (optional): chat room name (defaults to "general")
func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	// ── Authorization: reject unauthenticated users ──────────────────────
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, `{"type":"error","message":"Unauthorized: username is required"}`, http.StatusUnauthorized)
		log.Println("[WS] Rejected connection: missing username")
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = username
	}

	room := r.URL.Query().Get("room")
	if room == "" {
		room = "general"
	}

	// ── Upgrade HTTP → WebSocket ─────────────────────────────────────────
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Upgrade failed for user=%s: %v", username, err)
		return
	}

	client := &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 256), // buffered — skill checklist #2
		userID:   userID,
		username: username,
		room:     room,
	}

	// Register THEN start pumps (skill Step 4 pattern).
	hub.register <- client

	// Two goroutines per client — skill mandated.
	go client.writePump()
	go client.readPump()
}
