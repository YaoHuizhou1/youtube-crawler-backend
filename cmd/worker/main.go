package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/example/youtube-dialogue-crawler/internal/config"
	"github.com/example/youtube-dialogue-crawler/internal/ml"
	"github.com/example/youtube-dialogue-crawler/internal/pkg/logger"
	"github.com/example/youtube-dialogue-crawler/internal/pkg/websocket"
	"github.com/example/youtube-dialogue-crawler/internal/repository"
	"github.com/example/youtube-dialogue-crawler/internal/service"
	"github.com/example/youtube-dialogue-crawler/internal/worker"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger.Init(cfg.Server.Mode)
	defer logger.Sync()

	// Connect to PostgreSQL
	db, err := pgxpool.New(context.Background(), cfg.Database.URL)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(context.Background()); err != nil {
		logger.Fatalf("Failed to ping database: %v", err)
	}
	logger.Info("Connected to PostgreSQL")

	// Connect to ML service
	var mlClient *ml.Client
	if cfg.ML.ServiceAddr != "" {
		mlClient, err = ml.NewClient(cfg.ML.ServiceAddr)
		if err != nil {
			logger.Warnf("Failed to connect to ML service: %v (continuing without ML)", err)
		} else {
			defer mlClient.Close()
			logger.Info("Connected to ML service")
		}
	}

	// Create repositories
	taskRepo := repository.NewTaskRepository(db)
	videoRepo := repository.NewVideoRepository(db)
	tagRepo := repository.NewTagRepository(db)

	// Create services
	youtubeService := service.NewYouTubeService(cfg.YouTube.APIKey)
	discoveryService := service.NewDiscoveryService(youtubeService, taskRepo, videoRepo)
	analyzerService := service.NewAnalyzerService(videoRepo, mlClient)
	taggerService := service.NewTaggerService(videoRepo, tagRepo, cfg.Claude.APIKey)

	// Create WebSocket hub (for broadcasting)
	hub := websocket.NewHub()
	go hub.Run()

	// Create Asynq client and server
	redisAddr := strings.TrimPrefix(cfg.Redis.URL, "redis://")
	asynqClient := worker.NewClient(redisAddr)
	defer asynqClient.Close()

	srv := worker.NewServer(redisAddr, cfg.Worker.Concurrency)

	// Create task handlers
	discoveryHandler := worker.NewDiscoveryTaskHandler(discoveryService, taskRepo, hub, asynqClient)
	analysisHandler := worker.NewAnalysisTaskHandler(analyzerService, videoRepo, taskRepo, hub, asynqClient)
	taggingHandler := worker.NewTaggingTaskHandler(taggerService, hub)

	// Register handlers
	mux := asynq.NewServeMux()
	mux.HandleFunc(worker.TaskTypeDiscovery, discoveryHandler.ProcessTask)
	mux.HandleFunc(worker.TaskTypeAnalysis, analysisHandler.ProcessTask)
	mux.HandleFunc(worker.TaskTypeTagging, taggingHandler.ProcessTask)

	// Start worker in goroutine
	go func() {
		logger.Infof("Starting worker with concurrency %d", cfg.Worker.Concurrency)
		if err := srv.Run(mux); err != nil {
			logger.Fatalf("Failed to start worker: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down worker...")
	srv.Shutdown()
	logger.Info("Worker exited properly")
}
