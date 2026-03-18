package worker

import (
	"context"
	"encoding/json"

	"github.com/example/youtube-dialogue-crawler/internal/pkg/logger"
	"github.com/example/youtube-dialogue-crawler/internal/pkg/websocket"
	"github.com/example/youtube-dialogue-crawler/internal/service"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

type TaggingTaskHandler struct {
	tagger *service.TaggerService
	hub    *websocket.Hub
}

func NewTaggingTaskHandler(tagger *service.TaggerService, hub *websocket.Hub) *TaggingTaskHandler {
	return &TaggingTaskHandler{
		tagger: tagger,
		hub:    hub,
	}
}

type TaggingPayload struct {
	VideoID uuid.UUID `json:"video_id"`
}

func (h *TaggingTaskHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload TaggingPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}

	logger.Infof("Processing tagging task for video: %s", payload.VideoID)

	// Run tagging
	result, err := h.tagger.TagVideo(ctx, payload.VideoID)
	if err != nil {
		logger.Errorf("Tagging error: %v", err)
		return err
	}

	// Broadcast result
	h.hub.Broadcast("video_tagged", map[string]interface{}{
		"video_id":  payload.VideoID,
		"tag_count": len(result.Tags),
	})

	logger.Infof("Tagging completed for video: %s, generated %d tags", payload.VideoID, len(result.Tags))

	return nil
}

func CreateTaggingTask(videoID uuid.UUID) (*asynq.Task, error) {
	payload, err := json.Marshal(TaggingPayload{
		VideoID: videoID,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskTypeTagging, payload), nil
}
