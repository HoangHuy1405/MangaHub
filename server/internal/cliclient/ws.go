package cliclient

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"mangahub/pkg/models"
)

// WSClient manages a WebSocket connection to the Chat Server.
// Follow the same architecture as TCPClient and UDPClient.
type WSClient struct {
	cfg      *CLIConfig
	conn     *websocket.Conn
	done     chan struct{}
	mu       sync.Mutex // protects conn writes
	username string
	room     string
}

// NewWSClient creates a WSClient (not yet connected).
func NewWSClient(cfg *CLIConfig) *WSClient {
	return &WSClient{
		cfg:  cfg,
		done: make(chan struct{}),
	}
}

// Connect dials the WebSocket chat server and performs the join handshake.
//
// Authentication: the username is sent as a query parameter during the
// HTTP upgrade request. The server rejects connections without a username.
func (w *WSClient) Connect(username, room string) error {
	if username == "" {
		return fmt.Errorf("username is required for WebSocket chat")
	}
	if room == "" {
		room = "general"
	}
	w.username = username
	w.room = room

	// Build the WS URL with query parameters for auth and room selection.
	u, err := url.Parse(w.cfg.WSURL())
	if err != nil {
		return fmt.Errorf("invalid WebSocket URL: %w", err)
	}
	q := u.Query()
	q.Set("username", username)
	q.Set("user_id", username)
	q.Set("room", room)
	u.RawQuery = q.Encode()

	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}
	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("cannot connect to WebSocket chat server at %s — is api-server running?", w.cfg.WSURL())
	}
	w.conn = conn
	return nil
}

// readLoop runs in a background goroutine, reading messages from the server
// and printing them to the terminal. It closes the done channel when finished.
//
// Best Practice — Mistake #6 "Goroutine Leaks":
// The caller passes a context; when cancelled, the conn is closed which
// unblocks ReadMessage().
func (w *WSClient) readLoop() {
	defer close(w.done)

	for {
		_, rawMsg, err := w.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				fmt.Fprintf(os.Stderr, "\nConnection error: %v\n", err)
			}
			return
		}

		var msg models.ChatMessage
		if err := json.Unmarshal(rawMsg, &msg); err != nil {
			continue
		}

		w.renderMessage(msg)
	}
}

// renderMessage formats and prints a ChatMessage to the terminal.
func (w *WSClient) renderMessage(msg models.ChatMessage) {
	ts := time.Unix(msg.Timestamp, 0).Format("15:04")

	switch msg.Type {
	case models.MsgChat:
		fmt.Printf("[%s] %s: %s\n", ts, msg.Username, msg.Message)
	case models.MsgPM:
		if msg.Username == w.username {
			fmt.Printf("[%s] (PM to %s): %s\n", ts, msg.Target, msg.Message)
		} else {
			fmt.Printf("[%s] (PM from %s): %s\n", ts, msg.Username, msg.Message)
		}
	case models.MsgJoin:
		fmt.Printf("[%s] ● %s\n", ts, msg.Message)
	case models.MsgLeave:
		fmt.Printf("[%s] ○ %s\n", ts, msg.Message)
	case models.MsgSystem:
		fmt.Printf("%s\n", msg.Message)
	case models.MsgUsers:
		count := 0
		if msg.Message != "No users online" {
			count = strings.Count(msg.Message, "\n") + 1
		}
		fmt.Printf("\nOnline Users (%d):\n%s\n\n", count, msg.Message)
	case models.MsgHistory:
		w.renderHistory(msg.Message)
	case models.MsgError:
		fmt.Printf("✗ Error: %s\n", msg.Message)
	default:
		fmt.Printf("[%s] %s\n", ts, msg.Message)
	}

	// Re-print the prompt after receiving a message.
	fmt.Printf("%s> ", w.username)
}

// renderHistory parses and prints the history JSON payload.
func (w *WSClient) renderHistory(payload string) {
	var history []models.ChatMessage
	if err := json.Unmarshal([]byte(payload), &history); err != nil {
		fmt.Printf("\nRecent messages: (none)\n\n")
		return
	}
	if len(history) == 0 {
		fmt.Printf("\nRecent messages: (none)\n\n")
		return
	}
	fmt.Printf("\nRecent messages:\n")
	for _, m := range history {
		ts := time.Unix(m.Timestamp, 0).Format("15:04")
		fmt.Printf("  [%s] %s: %s\n", ts, m.Username, m.Message)
	}
	fmt.Println()
}

// Listen enters interactive chat mode: reads stdin and prints incoming
// messages. Blocks until the user types /quit, the context is cancelled,
// or the connection drops.
//
// Best Practice — Mistake #6: "Goroutine Leaks"
// Accept a context so the caller can signal shutdown (Ctrl+C) gracefully.
func (w *WSClient) Listen(ctx context.Context) error {
	if w.conn == nil {
		return fmt.Errorf("not connected — call Connect first")
	}

	// Start background reader goroutine.
	go w.readLoop()

	// Cancel-aware goroutine: close conn when context is done.
	go func() {
		<-ctx.Done()
		w.Close()
	}()

	// Print the interactive mode banner.
	w.printBanner()

	// Interactive stdin loop.
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("%s> ", w.username)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			fmt.Printf("%s> ", w.username)
			continue
		}

		// Handle slash commands.
		if strings.HasPrefix(line, "/") {
			if w.handleCommand(line) {
				return nil // /quit
			}
			fmt.Printf("%s> ", w.username)
			continue
		}

		// Send chat message.
		if err := w.SendMessage(line); err != nil {
			fmt.Fprintf(os.Stderr, "✗ Send failed: %v\n", err)
		}
		// Don't re-print prompt here — readLoop will print it after echo.
	}

	return nil
}

// handleCommand processes slash commands. Returns true if the user wants to quit.
func (w *WSClient) handleCommand(cmd string) bool {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return false
	}

	switch parts[0] {
	case "/quit", "/exit":
		fmt.Println("\nLeaving chat...")
		w.Close()
		fmt.Println("✓ Disconnected from chat server")
		return true

	case "/help":
		fmt.Println("\nChat Commands:")
		fmt.Println("  /help            - Show this help")
		fmt.Println("  /users           - List online users")
		fmt.Println("  /quit            - Leave chat")
		fmt.Println("  /pm <user> <msg> - Private message")
		fmt.Println("  /manga <id>      - Switch to manga chat")
		fmt.Println("  /history         - Show recent history")
		fmt.Println("  /status          - Connection status")
		fmt.Println()

	case "/users":
		w.sendTyped(models.MsgUsers, "")

	case "/history":
		w.sendTyped(models.MsgHistory, "")

	case "/pm":
		if len(parts) < 3 {
			fmt.Println("Usage: /pm <user> <msg>")
			return false
		}
		target := parts[1]
		message := strings.Join(parts[2:], " ")
		pm := models.ChatMessage{
			Type:      models.MsgPM,
			UserID:    w.username,
			Username:  w.username,
			Target:    target,
			Message:   message,
			Timestamp: time.Now().Unix(),
		}
		w.mu.Lock()
		w.conn.WriteJSON(pm)
		w.mu.Unlock()

	case "/manga":
		if len(parts) < 2 {
			fmt.Println("Usage: /manga <id>")
			return false
		}
		mangaID := parts[1]
		fmt.Printf("Switching to manga chat: %s...\n", mangaID)
		w.Close()
		err := w.Connect(w.username, mangaID)
		if err != nil {
			fmt.Printf("✗ Failed to switch room: %v\n", err)
			return true // quit on failure
		}
		// Restart readLoop
		w.done = make(chan struct{})
		go w.readLoop()
		w.printBanner()

	case "/status":
		fmt.Printf("\nConnection Status:\n")
		fmt.Printf("  Server:   %s\n", w.cfg.WSURL())
		fmt.Printf("  User:     %s\n", w.username)
		fmt.Printf("  Room:     %s\n", w.room)
		fmt.Println()

	default:
		fmt.Printf("Unknown command: %s (type /help for available commands)\n", parts[0])
	}

	return false
}

// SendMessage sends a chat message to the server.
func (w *WSClient) SendMessage(text string) error {
	return w.sendTyped(models.MsgChat, text)
}

// sendTyped sends a message with the given type.
func (w *WSClient) sendTyped(msgType models.MessageType, text string) error {
	if w.conn == nil {
		return fmt.Errorf("not connected")
	}
	msg := models.ChatMessage{
		Type:      msgType,
		UserID:    w.username,
		Username:  w.username,
		Message:   text,
		Room:      w.room,
		Timestamp: time.Now().Unix(),
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.WriteJSON(msg)
}

// Close cleanly shuts down the WebSocket connection.
// Sends a CloseMessage first to avoid error code 1006 (skill Anti-Pattern #4).
func (w *WSClient) Close() {
	if w.conn != nil {
		// Send a clean close handshake.
		w.mu.Lock()
		w.conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
		w.mu.Unlock()
		w.conn.Close()
		w.conn = nil
	}
}

// IsConnected returns true when an active WS connection exists.
func (w *WSClient) IsConnected() bool {
	return w.conn != nil
}

// printBanner prints the interactive chat mode header.
func (w *WSClient) printBanner() {
	roomDisplay := "#" + w.room
	if w.room != "general" {
		roomDisplay = "#" + w.room + " (Manga Discussion)"
	}

	fmt.Println()
	fmt.Printf("✓ Connected to Chat\n")
	fmt.Printf("  Chat Room: %s\n", roomDisplay)
	fmt.Printf("  Your status: Online\n")
	fmt.Println()
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println("You are now in chat. Type your message and press Enter.")
	fmt.Println("Type /help for commands or /quit to leave.")
	fmt.Println()
}
