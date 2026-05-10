package websocket

import (
	"context"
	"log"
	"net/http"
	"time"
)

// ChatServer wraps the Hub + HTTP server for the WebSocket chat endpoint.
// It exposes Start()/Stop() so cmd/api-server can manage it inline,
// following the same pattern as udp.NotificationServer.
type ChatServer struct {
	hub  *Hub
	port string
	srv  *http.Server
}

// NewChatServer creates a ready-to-use ChatServer.
// Call Start() to begin accepting WebSocket connections.
func NewChatServer(port string) *ChatServer {
	hub := NewHub()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWS(hub, w, r)
	})

	return &ChatServer{
		hub:  hub,
		port: port,
		srv: &http.Server{
			Addr:    ":" + port,
			Handler: mux,
		},
	}
}

// Start launches the Hub goroutine and HTTP listener.
// It blocks until the server is shut down or a fatal error occurs.
//
// Skill checklist #1: go hub.Run() is launched BEFORE accepting connections.
func (cs *ChatServer) Start() error {
	go cs.hub.Run() // MUST be before ListenAndServe

	log.Printf("[WS] Chat Server listening on :%s", cs.port)
	if err := cs.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Stop gracefully shuts down the HTTP server with a 5-second deadline.
func (cs *ChatServer) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := cs.srv.Shutdown(ctx); err != nil {
		log.Printf("[WS] Shutdown error: %v", err)
	}
	log.Println("[WS] Chat Server stopped")
}
