package handlers

import (
	"context"

	"github.com/example/youtube-dialogue-crawler/internal/pkg/response"
	"github.com/example/youtube-dialogue-crawler/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StatsHandler struct {
	db        *pgxpool.Pool
	taskRepo  *repository.TaskRepository
	videoRepo *repository.VideoRepository
	tagRepo   *repository.TagRepository
}

func NewStatsHandler(db *pgxpool.Pool, taskRepo *repository.TaskRepository, videoRepo *repository.VideoRepository, tagRepo *repository.TagRepository) *StatsHandler {
	return &StatsHandler{db: db, taskRepo: taskRepo, videoRepo: videoRepo, tagRepo: tagRepo}
}

type OverviewStats struct {
	TotalTasks       int64 `json:"total_tasks"`
	RunningTasks     int64 `json:"running_tasks"`
	CompletedTasks   int64 `json:"completed_tasks"`
	TotalVideos      int64 `json:"total_videos"`
	DialogueVideos   int64 `json:"dialogue_videos"`
	AnalyzedVideos   int64 `json:"analyzed_videos"`
	PendingAnalysis  int64 `json:"pending_analysis"`
	ReviewedVideos   int64 `json:"reviewed_videos"`
}

func (h *StatsHandler) Overview(c *gin.Context) {
	ctx := c.Request.Context()
	stats := &OverviewStats{}

	// Task stats
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM tasks`).Scan(&stats.TotalTasks)
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM tasks WHERE status = 'running'`).Scan(&stats.RunningTasks)
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM tasks WHERE status = 'completed'`).Scan(&stats.CompletedTasks)

	// Video stats
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM videos`).Scan(&stats.TotalVideos)
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM videos WHERE is_dialogue = true`).Scan(&stats.DialogueVideos)
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM videos WHERE analysis_status = 'completed'`).Scan(&stats.AnalyzedVideos)
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM videos WHERE analysis_status = 'pending'`).Scan(&stats.PendingAnalysis)
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM videos WHERE reviewed = true`).Scan(&stats.ReviewedVideos)

	response.Success(c, stats)
}

type TaskStats struct {
	TaskID           uuid.UUID `json:"task_id"`
	TotalFound       int       `json:"total_found"`
	TotalAnalyzed    int       `json:"total_analyzed"`
	TotalConfirmed   int       `json:"total_confirmed"`
	DialogueCount    int64     `json:"dialogue_count"`
	NonDialogueCount int64     `json:"non_dialogue_count"`
	PendingCount     int64     `json:"pending_count"`
	ReviewedCount    int64     `json:"reviewed_count"`
}

func (h *StatsHandler) TaskStats(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid task ID")
		return
	}

	ctx := c.Request.Context()
	task, err := h.taskRepo.GetByID(ctx, id)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	if task == nil {
		response.NotFound(c, "Task not found")
		return
	}

	stats := &TaskStats{
		TaskID:         id,
		TotalFound:     task.TotalFound,
		TotalAnalyzed:  task.TotalAnalyzed,
		TotalConfirmed: task.TotalConfirmed,
	}

	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM videos WHERE task_id = $1 AND is_dialogue = true`, id).Scan(&stats.DialogueCount)
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM videos WHERE task_id = $1 AND is_dialogue = false`, id).Scan(&stats.NonDialogueCount)
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM videos WHERE task_id = $1 AND analysis_status = 'pending'`, id).Scan(&stats.PendingCount)
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM videos WHERE task_id = $1 AND reviewed = true`, id).Scan(&stats.ReviewedCount)

	response.Success(c, stats)
}

type TimelinePoint struct {
	Date         string `json:"date"`
	VideosFound  int64  `json:"videos_found"`
	Analyzed     int64  `json:"analyzed"`
	Dialogues    int64  `json:"dialogues"`
}

func (h *StatsHandler) Timeline(c *gin.Context) {
	ctx := c.Request.Context()
	days := 30 // Last 30 days

	rows, err := h.db.Query(ctx, `
		SELECT DATE(created_at) as date,
			   COUNT(*) as videos_found,
			   COUNT(*) FILTER (WHERE analysis_status = 'completed') as analyzed,
			   COUNT(*) FILTER (WHERE is_dialogue = true) as dialogues
		FROM videos
		WHERE created_at >= NOW() - INTERVAL '1 day' * $1
		GROUP BY DATE(created_at)
		ORDER BY date ASC
	`, days)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	defer rows.Close()

	var timeline []TimelinePoint
	for rows.Next() {
		var point TimelinePoint
		var date interface{}
		if err := rows.Scan(&date, &point.VideosFound, &point.Analyzed, &point.Dialogues); err != nil {
			continue
		}
		if d, ok := date.(string); ok {
			point.Date = d
		}
		timeline = append(timeline, point)
	}

	response.Success(c, timeline)
}

func (h *StatsHandler) TagStats(c *gin.Context) {
	stats, err := h.tagRepo.GetStats(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, stats)
}

// Ensure context import is used
var _ = context.Background
