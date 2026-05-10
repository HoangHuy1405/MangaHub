package websocket

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	"mangahub/pkg/models"
)

const maxHistoryPerRoom = 50

// Room represents a single chat room with its own client set and message history.
// Each room is independent — messages in #one-piece never leak to #general.
type Room struct {
	name    string
	clients map[*Client]bool
	history []models.ChatMessage
}

// newRoom creates an empty Room.
func newRoom(name string) *Room {
	return &Room{
		name:    name,
		clients: make(map[*Client]bool),
		history: make([]models.ChatMessage, 0, maxHistoryPerRoom),
	}
}

// addToHistory appends a message to the room's ring buffer,
// evicting the oldest entry when full.
func (r *Room) addToHistory(msg models.ChatMessage) {
	if len(r.history) >= maxHistoryPerRoom {
		r.history = r.history[1:]
	}
	r.history = append(r.history, msg)
}

// Hub is the central dispatcher that manages all WebSocket client connections
// using channels (no mutex). It runs as a single goroutine event loop.
//
// Design: gorilla/websocket Hub/Client pattern (skill: websocket-hub-client)
//   - rooms:      map of room name → *Room (each room owns its clients + history)
//   - broadcast:  channel for messages that must reach every client in a room
//   - register:   channel for new client join events
//   - unregister: channel for client leave/disconnect events
type Hub struct {
	rooms      map[string]*Room
	userMap    map[string]map[*Client]bool // Fast O(1) lookup for Private Messages
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

// NewHub creates a Hub ready to Run().
func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]*Room),
		userMap:    make(map[string]map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// getOrCreateRoom returns the Room for the given name, creating it if needed.
// MUST only be called from the Run() goroutine (single-threaded access).
func (h *Hub) getOrCreateRoom(name string) *Room {
	if r, ok := h.rooms[name]; ok {
		return r
	}
	r := newRoom(name)
	h.rooms[name] = r
	log.Printf("[WS] Room created: #%s", name)
	return r
}

// totalClients returns the count of all connected clients across all rooms.
func (h *Hub) totalClients() int {
	total := 0
	for _, r := range h.rooms {
		total += len(r.clients)
	}
	return total
}

// Run starts the single-goroutine event loop.
// MUST be called with `go hub.Run()` BEFORE accepting any connections.
func (h *Hub) Run() {
	log.Println("[WS] Hub event loop started")
	for {
		select {
		case client := <-h.register:
			room := h.getOrCreateRoom(client.room)
			room.clients[client] = true
			
			// Map user for O(1) PM routing
			if h.userMap[client.username] == nil {
				h.userMap[client.username] = make(map[*Client]bool)
			}
			h.userMap[client.username][client] = true

			log.Printf("[WS] Client registered: user=%s room=#%s (room=%d, total=%d)",
				client.username, client.room, len(room.clients), h.totalClients())

			// Broadcast a "join" notification to all clients in the same room.
			joinMsg := models.ChatMessage{
				Type:      models.MsgJoin,
				UserID:    client.userID,
				Username:  client.username,
				Message:   client.username + " joined the chat",
				Room:      client.room,
				Timestamp: time.Now().Unix(),
			}
			h.broadcastToRoom(joinMsg, room)

			// Send connected users count to the newly joined client.
			h.sendUserCount(client, room)

		case client := <-h.unregister:
			room, ok := h.rooms[client.room]
			if !ok {
				continue
			}
			if _, exists := room.clients[client]; !exists {
				continue
			}

			// Broadcast a "leave" notification before removing.
			leaveMsg := models.ChatMessage{
				Type:      models.MsgLeave,
				UserID:    client.userID,
				Username:  client.username,
				Message:   client.username + " left the chat",
				Room:      client.room,
				Timestamp: time.Now().Unix(),
			}

			delete(room.clients, client)
			
			// Clean up from userMap
			if userClients, ok := h.userMap[client.username]; ok {
				delete(userClients, client)
				if len(userClients) == 0 {
					delete(h.userMap, client.username)
				}
			}

			close(client.send)

			log.Printf("[WS] Client unregistered: user=%s room=#%s (room=%d, total=%d)",
				client.username, client.room, len(room.clients), h.totalClients())

			h.broadcastToRoom(leaveMsg, room)

			// Garbage-collect empty rooms (except "general" which always exists).
			if len(room.clients) == 0 && room.name != "general" {
				delete(h.rooms, room.name)
				log.Printf("[WS] Room destroyed (empty): #%s", room.name)
			}

		case msg := <-h.broadcast:
			// Parse the message to record in history and route by room.
			var chatMsg models.ChatMessage
			if err := json.Unmarshal(msg, &chatMsg); err == nil {
				if chatMsg.Type == models.MsgPM {
					targetClients, targetOnline := h.userMap[chatMsg.Target]

					// Send to Target in O(1)
					if targetOnline {
						for client := range targetClients {
							select {
							case client.send <- msg:
							default:
								log.Printf("[WS] Dropping slow client (PM target): user=%s", client.username)
								close(client.send)
							}
						}
					}

					// Send to Sender in O(1)
					if senderClients, senderOnline := h.userMap[chatMsg.Username]; senderOnline {
						for client := range senderClients {
							select {
							case client.send <- msg:
							default:
								log.Printf("[WS] Dropping slow client (PM sender): user=%s", client.username)
								close(client.send)
							}
						}
					}

					// If Target not online, send error to Sender
					if !targetOnline {
						errMsg := models.ChatMessage{
							Type:      models.MsgError,
							Message:   "User " + chatMsg.Target + " is not online",
							Timestamp: time.Now().Unix(),
						}
						errData, _ := json.Marshal(errMsg)
						if senderClients, senderOnline := h.userMap[chatMsg.Username]; senderOnline {
							for client := range senderClients {
								select {
								case client.send <- errData:
								default:
								}
							}
						}
					}
					continue
				}

				room := h.getOrCreateRoom(chatMsg.Room)

				// Store in history for /history command.
				if chatMsg.Type == models.MsgChat {
					room.addToHistory(chatMsg)
				}

				// Route to clients in this room only.
				for client := range room.clients {
					select {
					case client.send <- msg:
						// ok — message queued
					default:
						// Skill pattern: drop slow clients to prevent blocking.
						log.Printf("[WS] Dropping slow client: user=%s", client.username)
						close(client.send)
						delete(room.clients, client)
					}
				}
			}
		}
	}
}

// broadcastToRoom marshals a ChatMessage and sends it to all clients in the room.
func (h *Hub) broadcastToRoom(msg models.ChatMessage, room *Room) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[WS] broadcastToRoom marshal error: %v", err)
		return
	}
	for client := range room.clients {
		select {
		case client.send <- data:
		default:
			log.Printf("[WS] Dropping slow client during broadcast: user=%s", client.username)
			close(client.send)
			delete(room.clients, client)
		}
	}
}

// sendUserCount sends a "system" message to a single client with the current
// user count and names in their room.
func (h *Hub) sendUserCount(target *Client, room *Room) {
	var users []string
	for c := range room.clients {
		users = append(users, c.username)
	}
	sysMsg := models.ChatMessage{
		Type:      models.MsgSystem,
		Message:   formatUserList(users),
		Room:      room.name,
		Timestamp: time.Now().Unix(),
	}
	data, err := json.Marshal(sysMsg)
	if err != nil {
		return
	}
	select {
	case target.send <- data:
	default:
	}
}

// handleUsersRequest sends a "users" message listing all online users to the requesting client.
func (h *Hub) handleUsersRequest(client *Client) {
	var entries []string
	for _, room := range h.rooms {
		for c := range room.clients {
			roomName := room.name
			if roomName == "general" {
				roomName = "General Chat"
			} else {
				// Convert "one-piece" to "One Piece Discussion"
				// A simple title case logic
				words := strings.Split(roomName, "-")
				for i, w := range words {
					if len(w) > 0 {
						words[i] = strings.ToUpper(w[:1]) + w[1:]
					}
				}
				roomName = strings.Join(words, " ") + " Discussion"
			}
			entry := c.username + " (" + roomName + ")"
			entries = append(entries, entry)
		}
	}
	usersMsg := models.ChatMessage{
		Type:      models.MsgUsers,
		Message:   formatUserList(entries),
		Room:      client.room,
		Timestamp: time.Now().Unix(),
	}
	data, err := json.Marshal(usersMsg)
	if err != nil {
		return
	}
	select {
	case client.send <- data:
	default:
	}
}

// handleHistoryRequest sends the room-specific history to the requesting client.
func (h *Hub) handleHistoryRequest(client *Client) {
	room, ok := h.rooms[client.room]
	if !ok {
		// Room doesn't exist — send empty history.
		h.sendEmptyHistory(client)
		return
	}

	if len(room.history) == 0 {
		h.sendEmptyHistory(client)
		return
	}

	data, err := json.Marshal(room.history)
	if err != nil {
		return
	}
	historyMsg := models.ChatMessage{
		Type:      models.MsgHistory,
		Message:   string(data),
		Room:      client.room,
		Timestamp: time.Now().Unix(),
	}
	resp, err := json.Marshal(historyMsg)
	if err != nil {
		return
	}
	select {
	case client.send <- resp:
	default:
	}
}

// sendEmptyHistory sends an empty history response to a client.
func (h *Hub) sendEmptyHistory(client *Client) {
	historyMsg := models.ChatMessage{
		Type:      models.MsgHistory,
		Message:   "[]",
		Room:      client.room,
		Timestamp: time.Now().Unix(),
	}
	data, err := json.Marshal(historyMsg)
	if err != nil {
		return
	}
	select {
	case client.send <- data:
	default:
	}
}

// formatUserList formats a list of user names into a newline-separated string.
func formatUserList(users []string) string {
	if len(users) == 0 {
		return "No users online"
	}
	result := ""
	for i, u := range users {
		if i > 0 {
			result += "\n"
		}
		result += "● " + u
	}
	return result
}
