package service

import (
	"context"
	"encoding/json"

	"github.com/example/youtube-dialogue-crawler/internal/models"
	"github.com/example/youtube-dialogue-crawler/internal/pkg/logger"
	"github.com/example/youtube-dialogue-crawler/internal/repository"
	"github.com/google/uuid"
)

type DiscoveryService struct {
	youtube   *YouTubeService
	taskRepo  *repository.TaskRepository
	videoRepo *repository.VideoRepository
}

func NewDiscoveryService(youtube *YouTubeService, taskRepo *repository.TaskRepository, videoRepo *repository.VideoRepository) *DiscoveryService {
	return &DiscoveryService{
		youtube:   youtube,
		taskRepo:  taskRepo,
		videoRepo: videoRepo,
	}
}

func (s *DiscoveryService) RunDiscovery(ctx context.Context, taskID uuid.UUID) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return nil
	}

	var config models.TaskConfig
	if err := json.Unmarshal(task.Config, &config); err != nil {
		return err
	}

	// Set defaults
	if config.MaxResults == 0 {
		config.MaxResults = 50
	}
	if config.MinDuration == 0 {
		config.MinDuration = 300 // 5 minutes
	}
	if config.MaxDuration == 0 {
		config.MaxDuration = 10800 // 3 hours
	}

	var foundCount int
	var videoIDs []string

	switch task.Type {
	case models.TaskTypeKeywordSearch:
		videoIDs, err = s.searchByKeywords(ctx, config.Keywords, config.MaxResults)
	case models.TaskTypeChannelMonitor:
		videoIDs, err = s.searchByChannels(ctx, config.ChannelIDs, config.MaxResults)
	}

	if err != nil {
		logger.Errorf("Discovery error: %v", err)
		return err
	}

	// Get video details and filter
	if len(videoIDs) > 0 {
		videos, err := s.youtube.GetVideoDetails(ctx, videoIDs)
		if err != nil {
			logger.Errorf("Get video details error: %v", err)
			return err
		}

		for _, item := range videos {
			// Check if already exists
			exists, _ := s.videoRepo.Exists(ctx, item.ID)
			if exists {
				continue
			}

			video := s.youtube.ToVideo(item, &taskID)

			// Apply duration filter
			if video.DurationSeconds != nil {
				duration := *video.DurationSeconds
				if duration < config.MinDuration || duration > config.MaxDuration {
					continue
				}
			}

			if err := s.videoRepo.Create(ctx, video); err != nil {
				logger.Errorf("Create video error: %v", err)
				continue
			}

			foundCount++
		}
	}

	// Update task progress
	s.taskRepo.UpdateProgress(ctx, taskID, 100, task.TotalFound+foundCount, task.TotalAnalyzed, task.TotalConfirmed)

	return nil
}

func (s *DiscoveryService) searchByKeywords(ctx context.Context, keywords []string, maxResults int) ([]string, error) {
	var allVideoIDs []string

	for _, keyword := range keywords {
		result, err := s.youtube.Search(ctx, keyword, maxResults, "")
		if err != nil {
			logger.Warnf("Search keyword '%s' error: %v", keyword, err)
			continue
		}

		for _, item := range result.Items {
			if item.ID.VideoID != "" {
				allVideoIDs = append(allVideoIDs, item.ID.VideoID)
			}
		}
	}

	return allVideoIDs, nil
}

func (s *DiscoveryService) searchByChannels(ctx context.Context, channelIDs []string, maxResults int) ([]string, error) {
	var allVideoIDs []string

	for _, channelID := range channelIDs {
		result, err := s.youtube.GetChannelVideos(ctx, channelID, maxResults, "")
		if err != nil {
			logger.Warnf("Search channel '%s' error: %v", channelID, err)
			continue
		}

		for _, item := range result.Items {
			if item.ID.VideoID != "" {
				allVideoIDs = append(allVideoIDs, item.ID.VideoID)
			}
		}
	}

	return allVideoIDs, nil
}
