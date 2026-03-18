package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/example/youtube-dialogue-crawler/internal/models"
	"github.com/example/youtube-dialogue-crawler/internal/pkg/logger"
	"github.com/example/youtube-dialogue-crawler/internal/repository"
	"github.com/google/uuid"
)

type TaggerService struct {
	videoRepo   *repository.VideoRepository
	tagRepo     *repository.TagRepository
	claudeKey   string
	httpClient  *http.Client
}

func NewTaggerService(videoRepo *repository.VideoRepository, tagRepo *repository.TagRepository, claudeKey string) *TaggerService {
	return &TaggerService{
		videoRepo: videoRepo,
		tagRepo:   tagRepo,
		claudeKey: claudeKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

type TaggingResult struct {
	Tags     []models.VideoTag       `json:"tags"`
	Segments []models.DialogueSegment `json:"segments"`
}

type ClaudeRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	Messages    []ClaudeMessage `json:"messages"`
}

type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ClaudeResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

func (s *TaggerService) TagVideo(ctx context.Context, videoID uuid.UUID) (*TaggingResult, error) {
	video, err := s.videoRepo.GetByID(ctx, videoID)
	if err != nil {
		return nil, err
	}
	if video == nil {
		return nil, fmt.Errorf("video not found")
	}

	result := &TaggingResult{}

	// Generate content tags using Claude
	if s.claudeKey != "" {
		tags, err := s.generateTags(ctx, video)
		if err != nil {
			logger.Warnf("Tag generation error: %v", err)
		} else {
			result.Tags = tags
			if err := s.tagRepo.CreateBatch(ctx, videoID, tags); err != nil {
				logger.Warnf("Save tags error: %v", err)
			}
		}
	}

	// Add basic auto-tags based on metadata
	autoTags := s.generateAutoTags(video)
	for _, tag := range autoTags {
		tag.VideoID = videoID
		result.Tags = append(result.Tags, tag)
	}
	if err := s.tagRepo.CreateBatch(ctx, videoID, autoTags); err != nil {
		logger.Warnf("Save auto tags error: %v", err)
	}

	return result, nil
}

func (s *TaggerService) generateTags(ctx context.Context, video *models.Video) ([]models.VideoTag, error) {
	description := ""
	if video.Description != nil {
		description = *video.Description
		if len(description) > 1000 {
			description = description[:1000]
		}
	}

	prompt := fmt.Sprintf(`Analyze this YouTube video and generate content tags.

Title: %s
Description: %s

Generate tags in the following categories:
1. topic: Main topics discussed (e.g., "technology", "business", "health")
2. format: Content format (e.g., "interview", "podcast", "debate")
3. guest: Names of guests if identifiable from title/description

Return JSON array with format:
[{"tag_name": "tag", "tag_type": "topic|format|guest", "confidence": 0.0-1.0}]

Only return the JSON array, no other text.`, video.Title, description)

	reqBody := ClaudeRequest{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: 1000,
		Messages: []ClaudeMessage{
			{Role: "user", Content: prompt},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.claudeKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Claude API error: %s - %s", resp.Status, string(body))
	}

	var claudeResp ClaudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return nil, err
	}

	if len(claudeResp.Content) == 0 {
		return nil, fmt.Errorf("empty response from Claude")
	}

	// Parse tags from response
	var tagData []struct {
		TagName    string  `json:"tag_name"`
		TagType    string  `json:"tag_type"`
		Confidence float64 `json:"confidence"`
	}

	text := claudeResp.Content[0].Text
	// Find JSON array in response
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start != -1 && end != -1 && end > start {
		text = text[start : end+1]
	}

	if err := json.Unmarshal([]byte(text), &tagData); err != nil {
		return nil, fmt.Errorf("parse tags: %w", err)
	}

	var tags []models.VideoTag
	for _, t := range tagData {
		tagType := models.TagType(t.TagType)
		if tagType != models.TagTypeTopic && tagType != models.TagTypeFormat && tagType != models.TagTypeGuest {
			tagType = models.TagTypeTopic
		}
		tags = append(tags, models.VideoTag{
			TagName:    t.TagName,
			TagType:    tagType,
			Confidence: &t.Confidence,
			Source:     models.TagSourceLLM,
		})
	}

	return tags, nil
}

func (s *TaggerService) generateAutoTags(video *models.Video) []models.VideoTag {
	var tags []models.VideoTag
	titleLower := strings.ToLower(video.Title)

	// Format detection
	formatKeywords := map[string]string{
		"podcast":   "Podcast",
		"interview": "Interview",
		"debate":    "Debate",
		"q&a":       "Q&A",
		"talk":      "Talk Show",
		"chat":      "Conversation",
	}

	for keyword, tagName := range formatKeywords {
		if strings.Contains(titleLower, keyword) {
			conf := 0.9
			tags = append(tags, models.VideoTag{
				TagName:    tagName,
				TagType:    models.TagTypeFormat,
				Confidence: &conf,
				Source:     models.TagSourceAuto,
			})
			break
		}
	}

	return tags
}
