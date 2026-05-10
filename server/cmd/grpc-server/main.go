package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	grpcimpl "mangahub/internal/grpc"
	"mangahub/internal/repository"
	"mangahub/pkg/database"
	"mangahub/pkg/utils/config"
	pb "mangahub/proto/manga"
)

func main() {
	// Load Configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("[gRPC] Failed to load config: %v", err)
	}

	// Initialize Database — reuse same DB as HTTP API (DRY)
	db, err := database.InitDB(cfg.API_CONFIG.DB_PATH)
	if err != nil {
		log.Fatalf("[gRPC] Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Seed data (same as api-server — ensures data is available)
	if err := database.SeedManga(db, "data/manga_seed.json"); err != nil {
		log.Printf("[gRPC] Warning: Failed to seed manga data: %v", err)
	}

	// Repositories — same layer used by HTTP API
	mangaRepo := repository.NewMangaRepository(db)
	userRepo := repository.NewUserRepository(db)

	// Listen on gRPC port
	port := cfg.NETWORK_CONFIG.GRPC_PORT
	if port == "" {
		port = "9092"
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("[gRPC] Failed to listen on :%s: %v", port, err)
	}

	// Create gRPC server and register the MangaService implementation
	grpcServer := grpc.NewServer()
	pb.RegisterMangaServiceServer(grpcServer, grpcimpl.NewMangaServer(mangaRepo, userRepo))

	// Graceful shutdown on OS signal
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("[gRPC] Shutting down gracefully...")
		grpcServer.GracefulStop()
	}()

	log.Printf("[gRPC] MangaService listening on :%s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("[gRPC] Server failed: %v", err)
	}
}
