package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"mangahub/internal/tcp"
	"mangahub/pkg/utils/config"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("[TCP-MAIN] Failed to load config: %v", err)
	}

	port := cfg.NETWORK_CONFIG.TCP_PORT

	srv := tcp.NewProgressSyncServer(port)

	// ── Graceful shutdown via OS signal ──────────────────────────────────────
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("[TCP-MAIN] Starting Progress Sync Server on :%s", port)
		if err := srv.Start(); err != nil {
			log.Printf("[TCP-MAIN] Server exited with error: %v", err)
		}
	}()

	// Block until OS signal received
	sig := <-sigCh
	log.Printf("[TCP-MAIN] Received signal %s — shutting down gracefully", sig)
	srv.Stop()
	log.Println("[TCP-MAIN] Shutdown complete")
}
