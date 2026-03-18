package worker

import (
	"context"
	"encoding/json"

	"github.com/example/youtube-dialogue-crawler/internal/models"
	"github.com/example/youtube-dialogue-crawler/internal/pkg/logger"
	"github.com/example/youtube-dialogue-crawler/internal/pkg/websocket"
	"github.com/example/youtube-dialogue-crawler/internal/repository"
	"github.com/example/youtube-dialogue-crawler/internal/service"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

type AnalysisTaskHandler struct {
	analyzer  *service.AnalyzerService
	videoRepo *repository.VideoRepository
	taskRepo  *repository.TaskRepository
	hub       *websocket.Hub
	client    *asynq.Client
}

func NewAnalysisTaskHandler(
	analyzer *service.AnalyzerService,
	videoRepo *repository.VideoRepository,
	taskRepo *repository.TaskRepository,
	hub *websocket.Hub,
	client *asynq.Client,
) *AnalysisTaskHandler {
	return &AnalysisTaskHandler{
		analyzer:  analyzer,
		videoRepo: videoRepo,
		taskRepo:  taskRepo,
		hub:       hub,
		client:    client,
	}
}

type AnalysisPayload struct {
	VideoID uuid.UUID `json:"video_id"`
	TaskID  uuid.UUID `json:"task_id"`
}

func (h *AnalysisTaskHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload AnalysisPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}

	logger.Infof("Processing analysis task for video: %s", payload.VideoID)

	// Run analysis
	result, err := h.analyzer.AnalyzeVideo(ctx, payload.VideoID)
	if err != nil {
		logger.Errorf("Analysis error: %v", err)
		errMsg := err.Error()
		h.videoRepo.UpdateAnalysisStatus(ctx, payload.VideoID, models.AnalysisStatusFailed, &errMsg)
		return err
	}

	// Broadcast result
	h.hub.Broadcast("video_analyzed", map[string]interface{}{
		"video_id":    payload.VideoID,
		"is_dialogue": result.IsDialogue,
		"confidence":  result.Confidence,
	})

	// If confirmed as dialogue, queue tagging task
	if result.IsDialogue {
		taggingTask, _ := CreateTaggingTask(payload.VideoID)
		if taggingTask != nil {
			h.client.Enqueue(taggingTask, asynq.Queue("low"))
		}

		// Update task stats
		if payload.TaskID != uuid.Nil {
			task, _ := h.taskRepo.GetByID(ctx, payload.TaskID)
			if task != nil {
				h.taskRepo.UpdateProgress(ctx, payload.TaskID, task.Progress, task.TotalFound, task.TotalAnalyzed+1, task.TotalConfirmed+1)
			}
		}
	} else {
		// Update task stats
		if payload.TaskID != uuid.Nil {
			task, _ := h.taskRepo.GetByID(ctx, payload.TaskID)
			if task != nil {
				h.taskRepo.UpdateProgress(ctx, payload.TaskID, task.Progress, task.TotalFound, task.TotalAnalyzed+1, task.TotalConfirmed)
			}
		}
	}

	logger.Infof("Analysis completed for video: %s, is_dialogue: %v, confidence: %.2f",
		payload.VideoID, result.IsDialogue, result.Confidence)

	return nil
}

func CreateAnalysisTask(videoID, taskID uuid.UUID) (*asynq.Task, error) {
	payload, err := json.Marshal(AnalysisPayload{
		VideoID: videoID,
		TaskID:  taskID,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskTypeAnalysis, payload), nil
}
