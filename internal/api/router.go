package api

import (
	"github.com/example/youtube-dialogue-crawler/internal/api/handlers"
	"github.com/example/youtube-dialogue-crawler/internal/api/middleware"
	"github.com/example/youtube-dialogue-crawler/internal/pkg/websocket"
	"github.com/example/youtube-dialogue-crawler/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(db *pgxpool.Pool, asynqClient *asynq.Client, hub *websocket.Hub) *gin.Engine {
	router := gin.New()

	// Middleware
	router.Use(middleware.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Initialize repositories
	taskRepo := repository.NewTaskRepository(db)
	videoRepo := repository.NewVideoRepository(db)
	tagRepo := repository.NewTagRepository(db)

	// Initialize handlers
	taskHandler := handlers.NewTaskHandler(taskRepo, asynqClient, hub)
	videoHandler := handlers.NewVideoHandler(videoRepo, tagRepo, hub)
	statsHandler := handlers.NewStatsHandler(db, taskRepo, videoRepo, tagRepo)
	wsHandler := handlers.NewWebSocketHandler(hub)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Task routes
		tasks := v1.Group("/tasks")
		{
			tasks.POST("", taskHandler.Create)
			tasks.GET("", taskHandler.List)
			tasks.GET("/:id", taskHandler.Get)
			tasks.PUT("/:id", taskHandler.Update)
			tasks.DELETE("/:id", taskHandler.Delete)
			tasks.POST("/:id/start", taskHandler.Start)
			tasks.POST("/:id/pause", taskHandler.Pause)
			tasks.POST("/:id/stop", taskHandler.Stop)
		}

		// Video routes
		videos := v1.Group("/videos")
		{
			videos.GET("", videoHandler.List)
			videos.GET("/:id", videoHandler.Get)
			videos.GET("/:id/segments", videoHandler.GetSegments)
			videos.GET("/:id/tags", videoHandler.GetTags)
			videos.PUT("/:id/review", videoHandler.Review)
			videos.POST("/:id/tags", videoHandler.AddTag)
			videos.DELETE("/:id/tags/:tagId", videoHandler.DeleteTag)
			videos.POST("/export", videoHandler.Export)
		}

		// Stats routes
		stats := v1.Group("/stats")
		{
			stats.GET("/overview", statsHandler.Overview)
			stats.GET("/tasks/:id", statsHandler.TaskStats)
			stats.GET("/timeline", statsHandler.Timeline)
			stats.GET("/tags", statsHandler.TagStats)
		}
	}

	// WebSocket
	router.GET("/ws/notifications", wsHandler.Handle)

	return router
}
