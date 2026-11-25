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

	repository.SeedAdmin(context.Background(), db, cfg.AdminEmail, cfg.AdminPassword)

	userRepo := repository.NewUserRepository(db)
	eventRepo := repository.NewEventRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	auditLogRepo := repository.NewAuditLogRepository(db)

	authService := service.NewAuthService(userRepo, cfg)
	eventService := service.NewEventService(eventRepo)
	orderService := service.NewOrderService(orderRepo)

	authHandler := handler.NewAuthHandler(authService)
	eventHandler := handler.NewEventHandler(eventService)
	orderHandler := handler.NewOrderHandler(orderService)
	auditLogHandler := handler.NewAuditLogHandler(auditLogRepo)

	mux := http.NewServeMux()

	// Public Routes
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true, "message": "Service sehat", "data": {"status": "up", "database": "up"}}`))
	})
	mux.HandleFunc("POST /api/v1/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("GET /api/v1/events", eventHandler.ListPublic)
	mux.HandleFunc("GET /api/v1/events/{id}", eventHandler.GetDetail)

	// Auth Routes (Customer/Admin)
	authMux := http.NewServeMux()
	authMux.HandleFunc("GET /api/v1/auth/me", authHandler.Me)
	mux.Handle("/api/v1/auth/me", middleware.AuthMiddleware([]byte(cfg.JWTSecret), "")(authMux))

	// Customer Only Routes (Or logged in)
	customerMux := http.NewServeMux()
	customerMux.HandleFunc("POST /api/v1/orders", orderHandler.Create)
	customerMux.HandleFunc("POST /api/v1/orders/{id}/pay", orderHandler.Pay)
	customerMux.HandleFunc("POST /api/v1/orders/{id}/cancel", orderHandler.Cancel)
	customerMux.HandleFunc("GET /api/v1/orders", orderHandler.List)
	customerMux.HandleFunc("GET /api/v1/orders/{id}", orderHandler.Detail)
	mux.Handle("/api/v1/orders", middleware.AuthMiddleware([]byte(cfg.JWTSecret), "")(customerMux))
	mux.Handle("/api/v1/orders/", middleware.AuthMiddleware([]byte(cfg.JWTSecret), "")(customerMux))

	// Admin Only Routes
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("POST /api/v1/events", eventHandler.Create)
	adminMux.HandleFunc("PUT /api/v1/events/{id}", eventHandler.Update)
	adminMux.HandleFunc("PATCH /api/v1/events/{id}/publish", eventHandler.Publish)
	adminMux.HandleFunc("PATCH /api/v1/events/{id}/cancel", eventHandler.Cancel)
	adminMux.HandleFunc("GET /api/v1/orders", orderHandler.List)
	adminMux.HandleFunc("GET /api/v1/orders/{id}", orderHandler.Detail)
	adminMux.HandleFunc("GET /api/v1/audit-logs", auditLogHandler.List) // BARU
	mux.Handle("/api/v1/events", middleware.AuthMiddleware([]byte(cfg.JWTSecret), "admin")(adminMux))
	mux.Handle("/api/v1/events/", middleware.AuthMiddleware([]byte(cfg.JWTSecret), "admin")(adminMux))
	mux.Handle("/api/v1/audit-logs", middleware.AuthMiddleware([]byte(cfg.JWTSecret), "admin")(adminMux)) // BARU

	server := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: mux,
	}

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
