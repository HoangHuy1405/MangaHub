package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"mangahub/internal/udp"
	"mangahub/pkg/utils/config"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("[UDP-MAIN] Failed to load config: %v", err)
	}

	port := cfg.NETWORK_CONFIG.UDP_PORT

	srv := udp.NewNotificationServer(port)

	// ── Graceful shutdown via OS signal ──────────────────────────────────────
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("[UDP-MAIN] Starting Notification Server on :%s", port)
		if err := srv.Start(); err != nil {
			log.Printf("[UDP-MAIN] Server exited with error: %v", err)
		}
	}()

	// Block until OS signal received
	sig := <-sigCh
	log.Printf("[UDP-MAIN] Received signal %s — shutting down gracefully", sig)
	srv.Stop()
	log.Println("[UDP-MAIN] Shutdown complete")
}
