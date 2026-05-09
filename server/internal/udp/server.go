package udp

import (
	"fmt"
	"log"
	"net"
	"sync"
)

// NotificationServer is a UDP server that:
//  1. Listens for registration packets from clients that want to receive
//     chapter-release notifications.
//  2. Exposes BroadcastNotification() so external callers (e.g. the REST
//     API chapter hook) can push a Notification to all registered clients.
//
// UDP is connectionless — there is no Accept loop. A single *net.UDPConn
// receives all incoming datagrams via ReadFromUDP (ref: skills doc §3).
type NotificationServer struct {
	port string
	conn *net.UDPConn

	// clients holds the address of every registered client.
	// key: addr.String() ("ip:port") for O(1) lookup/delete.
	clients map[string]*net.UDPAddr
	mu      sync.RWMutex

	// quit is closed by Stop() to signal the read loop to exit.
	quit chan struct{}
}

// NewNotificationServer creates a ready-to-use NotificationServer.
// Call Start() to begin listening.
func NewNotificationServer(port string) *NotificationServer {
	return &NotificationServer{
		port:    port,
		clients: make(map[string]*net.UDPAddr),
		quit:    make(chan struct{}),
	}
}

// Start resolves the UDP address, opens the listener, and enters the
// datagram read loop. It blocks until Stop() is called.
func (s *NotificationServer) Start() error {
	addr, err := net.ResolveUDPAddr("udp", ":"+s.port)
	if err != nil {
		return fmt.Errorf("[UDP] resolve addr: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("[UDP] listen on :%s: %w", s.port, err)
	}
	s.conn = conn
	// Do NOT defer conn.Close() here — Stop() owns the close so that
	// BroadcastNotification() can still use s.conn after Start() returns.

	log.Printf("[UDP] Notification Server listening on :%s", s.port)

	// Safe UDP payload limit — avoids fragmentation on all networks.
	// (ref: skills doc §3 Mistake #4 — Oversized UDP Packets)
	buf := make([]byte, 508)

	for {
		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			// Distinguish intentional shutdown from real errors.
			select {
			case <-s.quit:
				log.Println("[UDP] Listener closed — shutting down read loop")
				return nil
			default:
				log.Printf("[UDP] ReadFromUDP error: %v", err)
				continue
			}
		}

		// Copy the slice before passing to a goroutine — buf will be
		// overwritten on the next iteration.
		data := make([]byte, n)
		copy(data, buf[:n])
		go s.handlePacket(data, clientAddr)
	}
}

// Stop gracefully shuts down the server.
//
// Fix for Bug #2: closing s.conn is REQUIRED to unblock ReadFromUDP.
// Closing the quit channel alone does not interrupt a syscall.
func (s *NotificationServer) Stop() {
	close(s.quit)
	if s.conn != nil {
		s.conn.Close() // unblocks the ReadFromUDP call in Start()
	}
	log.Println("[UDP] Server stopped")
}
