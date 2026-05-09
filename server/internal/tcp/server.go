package tcp

import (
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/google/uuid"

	"mangahub/pkg/models"
)

// ProgressSyncServer is a TCP server that receives reading-progress updates
// from client devices and broadcasts them to all other active connections
// that share the same user_id.
//
// Connection registry layout:
//
//	connections[userID][connID] = net.Conn
//
// This two-level map ensures broadcasts are scoped per-user, not global.
type ProgressSyncServer struct {
	port     string
	listener net.Listener

	// Two-level map: userID → connID → conn
	// Protected by mu for all reads and writes.
	connections map[string]map[string]net.Conn
	mu          sync.RWMutex

	// broadcast is a buffered channel fed by every handler goroutine.
	// broadcastLoop drains it and fans out to per-user connections.
	broadcast chan models.ProgressUpdate

	// quit is closed by Stop() to signal all goroutines to exit.
	quit chan struct{}
}

// NewProgressSyncServer creates a ready-to-use ProgressSyncServer.
// Call Start() to begin accepting connections.
func NewProgressSyncServer(port string) *ProgressSyncServer {
	return &ProgressSyncServer{
		port:        port,
		connections: make(map[string]map[string]net.Conn),
		broadcast:   make(chan models.ProgressUpdate, 256), // buffered — prevents handler block
		quit:        make(chan struct{}),
	}
}

// Start begins listening for TCP connections on the configured port.
// It blocks until Stop() is called or a fatal listen error occurs.
func (s *ProgressSyncServer) Start() error {
	ln, err := net.Listen("tcp", ":"+s.port)
	if err != nil {
		return fmt.Errorf("[TCP] listen on :%s: %w", s.port, err)
	}
	s.listener = ln
	defer ln.Close()

	// Launch the broadcast fan-out goroutine before accepting connections.
	go s.broadcastLoop()

	log.Printf("[TCP] Progress Sync Server listening on :%s", s.port)

	// accept incoming connection
	for {
		conn, err := ln.Accept()
		if err != nil {
			// Check whether we shut down intentionally.
			select {
			case <-s.quit:
				log.Println("[TCP] Listener closed — shutting down accept loop")
				return nil
			default:
				log.Printf("[TCP] Accept error: %v", err)
				continue
			}
		}

		connID := uuid.New().String()
		log.Printf("[TCP] New connection: connID=%s remote=%s", connID, conn.RemoteAddr())
		go s.handleConnection(connID, conn)
	}
}

// Stop gracefully shuts the server down.
// It closes the quit channel (signalling all goroutines) and closes the
// listener (unblocking the Accept call in Start).
func (s *ProgressSyncServer) Stop() {
	close(s.quit)
	if s.listener != nil {
		s.listener.Close()
	}
	log.Println("[TCP] Server stopped")
}

// addConn registers conn under (userID, connID).
func (s *ProgressSyncServer) addConn(userID, connID string, conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.connections[userID] == nil {
		s.connections[userID] = make(map[string]net.Conn)
	}
	s.connections[userID][connID] = conn
}

// removeConn removes connID from the registry. If the user has no more
// connections, the top-level user entry is also deleted.
func (s *ProgressSyncServer) removeConn(userID, connID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if conns, ok := s.connections[userID]; ok {
		delete(conns, connID)
		if len(conns) == 0 {
			delete(s.connections, userID)
		}
	}
}
