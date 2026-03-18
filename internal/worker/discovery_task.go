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

type DiscoveryTaskHandler struct {
	discovery *service.DiscoveryService
	taskRepo  *repository.TaskRepository
	hub       *websocket.Hub
	client    *asynq.Client
}

func NewDiscoveryTaskHandler(
	discovery *service.DiscoveryService,
	taskRepo *repository.TaskRepository,
	hub *websocket.Hub,
	client *asynq.Client,
) *DiscoveryTaskHandler {
	return &DiscoveryTaskHandler{
		discovery: discovery,
		taskRepo:  taskRepo,
		hub:       hub,
		client:    client,
	}
}

func (h *DiscoveryTaskHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	taskIDStr := string(t.Payload())
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		return err
	}

	logger.Infof("Processing discovery task: %s", taskID)

	// Check task status
	task, err := h.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil || task.Status != models.TaskStatusRunning {
		return nil
	}

	// Run discovery
	if err := h.discovery.RunDiscovery(ctx, taskID); err != nil {
		errMsg := err.Error()
		h.taskRepo.UpdateStatus(ctx, taskID, models.TaskStatusFailed, &errMsg)
		h.hub.Broadcast("task_failed", map[string]interface{}{
			"task_id": taskID,
			"error":   errMsg,
		})
		return err
	}

	// Get updated task
	task, _ = h.taskRepo.GetByID(ctx, taskID)

	// Queue analysis tasks for found videos
	if task.TotalFound > 0 {
		// In a real implementation, you would query pending videos and enqueue them
		h.hub.Broadcast("discovery_complete", map[string]interface{}{
			"task_id":     taskID,
			"videos_found": task.TotalFound,
		})
	}

	// Update task status to completed
	h.taskRepo.UpdateStatus(ctx, taskID, models.TaskStatusCompleted, nil)
	h.hub.Broadcast("task_completed", task)

	logger.Infof("Discovery task completed: %s, found %d videos", taskID, task.TotalFound)

	return nil
}

// CreateDiscoveryTask creates a new discovery task
func CreateDiscoveryTask(taskID uuid.UUID) (*asynq.Task, error) {
	payload, err := json.Marshal(taskID.String())
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskTypeDiscovery, payload), nil
}
