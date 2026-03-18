package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"

	"github.com/example/youtube-dialogue-crawler/internal/models"
	"github.com/example/youtube-dialogue-crawler/internal/pkg/response"
	"github.com/example/youtube-dialogue-crawler/internal/pkg/websocket"
	"github.com/example/youtube-dialogue-crawler/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type VideoHandler struct {
	videoRepo *repository.VideoRepository
	tagRepo   *repository.TagRepository
	hub       *websocket.Hub
}

func NewVideoHandler(videoRepo *repository.VideoRepository, tagRepo *repository.TagRepository, hub *websocket.Hub) *VideoHandler {
	return &VideoHandler{videoRepo: videoRepo, tagRepo: tagRepo, hub: hub}
}

func (h *VideoHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid video ID")
		return
	}

	video, err := h.videoRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	if video == nil {
		response.NotFound(c, "Video not found")
		return
	}

	// Load tags
	tags, err := h.tagRepo.GetByVideoID(c.Request.Context(), id)
	if err == nil {
		video.Tags = tags
	}

	// Load segments
	segments, err := h.tagRepo.GetSegmentsByVideoID(c.Request.Context(), id)
	if err == nil {
		video.Segments = segments
	}

	response.Success(c, video)
}

func (h *VideoHandler) List(c *gin.Context) {
	var params models.VideoListParams
	if err := c.ShouldBindQuery(&params); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	videos, total, err := h.videoRepo.List(c.Request.Context(), &params)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithPage(c, videos, params.Page, params.PageSize, total)
}

func (h *VideoHandler) GetSegments(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid video ID")
		return
	}

	segments, err := h.tagRepo.GetSegmentsByVideoID(c.Request.Context(), id)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, segments)
}

func (h *VideoHandler) GetTags(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid video ID")
		return
	}

	tags, err := h.tagRepo.GetByVideoID(c.Request.Context(), id)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, tags)
}

func (h *VideoHandler) Review(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid video ID")
		return
	}

	var req models.ReviewVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.videoRepo.UpdateReview(c.Request.Context(), id, req.Result); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	video, _ := h.videoRepo.GetByID(c.Request.Context(), id)
	h.hub.Broadcast("video_reviewed", video)
	response.Success(c, video)
}

func (h *VideoHandler) AddTag(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid video ID")
		return
	}

	var req models.CreateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	tag, err := h.tagRepo.Create(c.Request.Context(), id, &req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Created(c, tag)
}

func (h *VideoHandler) DeleteTag(c *gin.Context) {
	tagIDStr := c.Param("tagId")
	tagID, err := uuid.Parse(tagIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid tag ID")
		return
	}

	if err := h.tagRepo.Delete(c.Request.Context(), tagID); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.NoContent(c)
}

func (h *VideoHandler) Export(c *gin.Context) {
	var req models.ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	params := &models.VideoListParams{
		Page:       1,
		PageSize:   10000,
		TaskID:     req.TaskID,
		IsDialogue: req.IsDialogue,
		Tags:       req.Tags,
	}

	videos, _, err := h.videoRepo.List(c.Request.Context(), params)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	switch req.Format {
	case "json":
		c.Header("Content-Disposition", "attachment; filename=videos.json")
		c.Header("Content-Type", "application/json")
		c.JSON(200, videos)

	case "csv":
		c.Header("Content-Disposition", "attachment; filename=videos.csv")
		c.Header("Content-Type", "text/csv")

		writer := csv.NewWriter(c.Writer)
		defer writer.Flush()

		// Write header
		writer.Write([]string{
			"id", "youtube_id", "title", "channel_name", "duration_seconds",
			"view_count", "is_dialogue", "dialogue_confidence", "published_at",
		})

		// Write data
		for _, v := range videos {
			isDialogue := ""
			if v.IsDialogue != nil {
				isDialogue = fmt.Sprintf("%v", *v.IsDialogue)
			}
			confidence := ""
			if v.DialogueConfidence != nil {
				confidence = fmt.Sprintf("%.4f", *v.DialogueConfidence)
			}
			channelName := ""
			if v.ChannelName != nil {
				channelName = *v.ChannelName
			}
			duration := ""
			if v.DurationSeconds != nil {
				duration = fmt.Sprintf("%d", *v.DurationSeconds)
			}
			viewCount := ""
			if v.ViewCount != nil {
				viewCount = fmt.Sprintf("%d", *v.ViewCount)
			}
			publishedAt := ""
			if v.PublishedAt != nil {
				publishedAt = v.PublishedAt.Format("2006-01-02")
			}

			writer.Write([]string{
				v.ID.String(), v.YouTubeID, v.Title, channelName, duration,
				viewCount, isDialogue, confidence, publishedAt,
			})
		}

	default:
		response.BadRequest(c, "Invalid format")
	}
}

type VideoStats struct {
	TotalVideos      int64   `json:"total_videos"`
	DialogueVideos   int64   `json:"dialogue_videos"`
	AnalyzedVideos   int64   `json:"analyzed_videos"`
	PendingVideos    int64   `json:"pending_videos"`
	ReviewedVideos   int64   `json:"reviewed_videos"`
	AvgConfidence    float64 `json:"avg_confidence"`
}

func (h *VideoHandler) Stats(c *gin.Context) {
	// This would typically query aggregated stats from the database
	// For now, return placeholder
	stats := VideoStats{
		TotalVideos:    0,
		DialogueVideos: 0,
		AnalyzedVideos: 0,
		PendingVideos:  0,
		ReviewedVideos: 0,
		AvgConfidence:  0,
	}
	response.Success(c, stats)
}

// Ensure json import is used
var _ = json.Marshal
