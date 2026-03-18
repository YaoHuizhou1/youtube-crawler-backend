package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/example/youtube-dialogue-crawler/internal/ml"
	"github.com/example/youtube-dialogue-crawler/internal/models"
	"github.com/example/youtube-dialogue-crawler/internal/pkg/logger"
	"github.com/example/youtube-dialogue-crawler/internal/repository"
	"github.com/google/uuid"
)

type AnalyzerService struct {
	videoRepo *repository.VideoRepository
	mlClient  *ml.Client
}

func NewAnalyzerService(videoRepo *repository.VideoRepository, mlClient *ml.Client) *AnalyzerService {
	return &AnalyzerService{
		videoRepo: videoRepo,
		mlClient:  mlClient,
	}
}

type AnalysisResult struct {
	IsDialogue   bool    `json:"is_dialogue"`
	Confidence   float64 `json:"confidence"`
	VisualScore  float64 `json:"visual_score"`
	AudioScore   float64 `json:"audio_score"`
	MetaScore    float64 `json:"meta_score"`
	FaceCountAvg float64 `json:"face_count_avg"`
	SpeakerCount int     `json:"speaker_count"`
}

func (s *AnalyzerService) AnalyzeVideo(ctx context.Context, videoID uuid.UUID) (*AnalysisResult, error) {
	video, err := s.videoRepo.GetByID(ctx, videoID)
	if err != nil {
		return nil, err
	}
	if video == nil {
		return nil, fmt.Errorf("video not found")
	}

	// Update status to analyzing
	s.videoRepo.UpdateAnalysisStatus(ctx, videoID, models.AnalysisStatusAnalyzing, nil)

	result := &AnalysisResult{}

	// Step 1: Download video samples using yt-dlp
	videoPath, err := s.downloadVideoSamples(ctx, video.YouTubeID)
	if err != nil {
		errMsg := fmt.Sprintf("download error: %v", err)
		s.videoRepo.UpdateAnalysisStatus(ctx, videoID, models.AnalysisStatusFailed, &errMsg)
		return nil, err
	}
	defer cleanupTempFiles(videoPath)

	// Step 2: Visual analysis (face detection)
	if s.mlClient != nil {
		faceResult, err := s.mlClient.DetectFaces(ctx, videoPath)
		if err != nil {
			logger.Warnf("Face detection error: %v", err)
		} else {
			result.FaceCountAvg = faceResult.AvgFaceCount
			result.VisualScore = calculateVisualScore(faceResult.AvgFaceCount)
		}

		// Step 3: Audio analysis (speaker diarization)
		audioResult, err := s.mlClient.AnalyzeSpeakers(ctx, videoPath)
		if err != nil {
			logger.Warnf("Speaker analysis error: %v", err)
		} else {
			result.SpeakerCount = audioResult.SpeakerCount
			result.AudioScore = calculateAudioScore(audioResult.SpeakerCount, audioResult.DialogueRatio)
		}
	}

	// Step 4: Metadata analysis
	result.MetaScore = s.analyzeMetadata(video)

	// Step 5: Calculate final score
	// Weights: visual 40%, audio 40%, meta 20%
	result.Confidence = result.VisualScore*0.4 + result.AudioScore*0.4 + result.MetaScore*0.2
	result.IsDialogue = result.Confidence >= 0.6

	// Save results
	metadata, _ := json.Marshal(models.VideoMetadata{
		VisualScore: result.VisualScore,
		AudioScore:  result.AudioScore,
		MetaScore:   result.MetaScore,
	})

	err = s.videoRepo.UpdateAnalysis(
		ctx,
		videoID,
		result.IsDialogue,
		result.Confidence,
		result.FaceCountAvg,
		result.SpeakerCount,
		metadata,
	)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *AnalyzerService) downloadVideoSamples(ctx context.Context, youtubeID string) (string, error) {
	// Download first 60 seconds for analysis
	outputPath := fmt.Sprintf("/tmp/yt_%s.mp4", youtubeID)
	url := fmt.Sprintf("https://www.youtube.com/watch?v=%s", youtubeID)

	cmd := exec.CommandContext(ctx, "yt-dlp",
		"-f", "best[height<=720]",
		"--download-sections", "*0-60",
		"-o", outputPath,
		url,
	)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("yt-dlp error: %w", err)
	}

	return outputPath, nil
}

func (s *AnalyzerService) analyzeMetadata(video *models.Video) float64 {
	score := 0.0

	// Check title for dialogue indicators
	dialogueKeywords := []string{
		"interview", "podcast", "talk", "conversation", "chat", "discussion",
		"debate", "q&a", "with", "featuring", "guest", "episode",
		"访谈", "对话", "播客", "聊天", "讨论",
	}

	titleLower := strings.ToLower(video.Title)
	for _, keyword := range dialogueKeywords {
		if strings.Contains(titleLower, keyword) {
			score += 0.2
		}
	}

	// Check description
	if video.Description != nil {
		descLower := strings.ToLower(*video.Description)
		for _, keyword := range dialogueKeywords {
			if strings.Contains(descLower, keyword) {
				score += 0.1
			}
		}
	}

	// Normalize score
	if score > 1.0 {
		score = 1.0
	}

	return score
}

func calculateVisualScore(avgFaceCount float64) float64 {
	// Ideal is 2 faces for two-person dialogue
	if avgFaceCount >= 1.8 && avgFaceCount <= 2.5 {
		return 1.0
	}
	if avgFaceCount >= 1.5 && avgFaceCount <= 3.0 {
		return 0.8
	}
	if avgFaceCount >= 1.0 && avgFaceCount <= 4.0 {
		return 0.5
	}
	return 0.2
}

func calculateAudioScore(speakerCount int, dialogueRatio float64) float64 {
	score := 0.0

	// Two speakers is ideal
	if speakerCount == 2 {
		score = 1.0
	} else if speakerCount == 1 || speakerCount == 3 {
		score = 0.5
	} else {
		score = 0.2
	}

	// Factor in dialogue ratio (how much of the audio has multiple speakers)
	score = score * (0.5 + dialogueRatio*0.5)

	return score
}

func cleanupTempFiles(path string) {
	exec.Command("rm", "-f", path).Run()
	// Also remove audio extract
	exec.Command("rm", "-f", strings.Replace(path, ".mp4", ".wav", 1)).Run()
}

// Ensure strconv import is used
var _ = strconv.Itoa
