package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"project-ticketing-golang/internal/config"
	"project-ticketing-golang/internal/handler"
	"project-ticketing-golang/internal/middleware"
	"project-ticketing-golang/internal/repository"
	"project-ticketing-golang/internal/service"
)

func main() {
	cfg := config.LoadConfig()

	db, err := repository.NewPostgresDB(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode)
	if err != nil {
		log.Fatalf("Gagal koneksi ke database: %v", err)
	}
	defer db.Close()
	log.Println("Berhasil terhubung ke database")

	// Panggil Seeder Admin
	repository.SeedAdmin(context.Background(), db, cfg.AdminEmail, cfg.AdminPassword)

	// Dependency Injection
	userRepo := repository.NewUserRepository(db)
	eventRepo := repository.NewEventRepository(db)

	authService := service.NewAuthService(userRepo, cfg)
	eventService := service.NewEventService(eventRepo)

	authHandler := handler.NewAuthHandler(authService)
	eventHandler := handler.NewEventHandler(eventService)

	// Setup Router
	mux := http.NewServeMux()

	// Public Routes
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true, "message": "Service sehat", "data": {"status": "up", "database": "up"}}`))
	})
	mux.HandleFunc("POST /api/v1/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)

	// Customer / Logged in Routes
	authMux := http.NewServeMux()
	authMux.HandleFunc("GET /api/v1/auth/me", authHandler.Me)
	mux.Handle("/api/v1/auth/me", middleware.AuthMiddleware([]byte(cfg.JWTSecret), "")(authMux))

	// Admin Only Routes
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("POST /api/v1/events", eventHandler.Create)
	adminMux.HandleFunc("PUT /api/v1/events/{id}", eventHandler.Update)
	adminMux.HandleFunc("PATCH /api/v1/events/{id}/publish", eventHandler.Publish)

	mux.Handle("/api/v1/events", middleware.AuthMiddleware([]byte(cfg.JWTSecret), "admin")(adminMux))
	mux.Handle("/api/v1/events/", middleware.AuthMiddleware([]byte(cfg.JWTSecret), "admin")(adminMux))

	server := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: mux,
	}

	// Graceful Shutdown
	go func() {
		log.Printf("Server berjalan di port %s", cfg.AppPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Mematikan server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	fmt.Println("Server keluar dengan rapi")
}
