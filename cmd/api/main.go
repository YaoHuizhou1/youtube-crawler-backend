package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/example/youtube-dialogue-crawler/internal/api"
	"github.com/example/youtube-dialogue-crawler/internal/config"
	"github.com/example/youtube-dialogue-crawler/internal/pkg/logger"
	"github.com/example/youtube-dialogue-crawler/internal/pkg/websocket"
	"github.com/example/youtube-dialogue-crawler/internal/worker"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger.Init(cfg.Server.Mode)
	defer logger.Sync()

	// Set Gin mode
	gin.SetMode(cfg.Server.Mode)

	// Connect to PostgreSQL
	db, err := pgxpool.New(context.Background(), cfg.Database.URL)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(context.Background()); err != nil {
		logger.Fatalf("Failed to ping database: %v", err)
	}
	logger.Info("Connected to PostgreSQL")

	// Create Asynq client
	redisAddr := strings.TrimPrefix(cfg.Redis.URL, "redis://")
	asynqClient := worker.NewClient(redisAddr)
	defer asynqClient.Close()

	// Create WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Create router
	router := api.NewRouter(db, asynqClient, hub)

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Infof("Starting API server on port %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited properly")
}

// Ensure fmt import is used
var _ = fmt.Sprintf
