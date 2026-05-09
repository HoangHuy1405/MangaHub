package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mangahub/internal/api-server/handlers"
	"mangahub/internal/api-server/routes"
	"mangahub/internal/repository"
	"mangahub/internal/service"
	"mangahub/pkg/database"
	"mangahub/pkg/utils/config"
)

func main() {
	// Load Configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("[MAIN] Failed to load config: %v", err)
	}

	// Initialize Database Connection
	db, err := database.InitDB(cfg.API_CONFIG.DB_PATH)
	if err != nil {
		log.Fatalf("[MAIN] Failed to initialize database: %v", err)
	}
	defer db.Close()

	if err := database.SeedManga(db, "data/manga_seed.json"); err != nil {
		log.Printf("[MAIN] Warning: Failed to seed manga data: %v", err)
	}

	// Repositories
	mangaRepo := repository.NewMangaRepository(db)
	userRepo := repository.NewUserRepository(db)
	authRepo := repository.NewAuthRepository(db)

	// Services
	mangaSvc := service.NewMangaService(mangaRepo)
	userSvc := service.NewUserService(userRepo, mangaRepo)
	authSvc := service.NewAuthService(authRepo, cfg.API_CONFIG.JWT_SECRET)

	// Handlers
	mangaHandler := handlers.NewMangaHandler(mangaSvc)
	userHandler := handlers.NewUserHandler(userSvc)
	authHandler := handlers.NewAuthHandler(authSvc)

	// Setup Router
	r := routes.SetupRouter(cfg, authHandler, mangaHandler, userHandler)

	// Server setup
	port := cfg.API_CONFIG.API_PORT
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Start server in a goroutine so it doesn't block
	go func() {
		log.Printf("[MAIN] Starting MangaHub API Server on :%s", port)
		log.Printf("[MAIN] API Base URL: http://localhost:%s/api/v1", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[MAIN] Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[MAIN] Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("[MAIN] Server forced to shutdown: %v", err)
	}

	log.Println("[MAIN] Server exiting gracefully")
}
