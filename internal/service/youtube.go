package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/example/youtube-dialogue-crawler/internal/models"
	"github.com/google/uuid"
)

type YouTubeService struct {
	apiKey     string
	httpClient *http.Client
}

func NewYouTubeService(apiKey string) *YouTubeService {
	return &YouTubeService{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type SearchResult struct {
	NextPageToken string        `json:"nextPageToken"`
	Items         []SearchItem  `json:"items"`
	PageInfo      PageInfo      `json:"pageInfo"`
}

type SearchItem struct {
	ID      ItemID  `json:"id"`
	Snippet Snippet `json:"snippet"`
}

type ItemID struct {
	Kind    string `json:"kind"`
	VideoID string `json:"videoId"`
}

type Snippet struct {
	PublishedAt  string     `json:"publishedAt"`
	ChannelID    string     `json:"channelId"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	ChannelTitle string     `json:"channelTitle"`
	Thumbnails   Thumbnails `json:"thumbnails"`
}

type Thumbnails struct {
	Default  Thumbnail `json:"default"`
	Medium   Thumbnail `json:"medium"`
	High     Thumbnail `json:"high"`
	Standard Thumbnail `json:"standard,omitempty"`
	Maxres   Thumbnail `json:"maxres,omitempty"`
}

type Thumbnail struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type PageInfo struct {
	TotalResults   int `json:"totalResults"`
	ResultsPerPage int `json:"resultsPerPage"`
}

type VideoListResult struct {
	Items []VideoItem `json:"items"`
}

type VideoItem struct {
	ID             string         `json:"id"`
	Snippet        Snippet        `json:"snippet"`
	ContentDetails ContentDetails `json:"contentDetails"`
	Statistics     Statistics     `json:"statistics"`
}

type ContentDetails struct {
	Duration string `json:"duration"`
}

type Statistics struct {
	ViewCount    string `json:"viewCount"`
	LikeCount    string `json:"likeCount"`
	CommentCount string `json:"commentCount"`
}

func (s *YouTubeService) Search(ctx context.Context, query string, maxResults int, pageToken string) (*SearchResult, error) {
	params := url.Values{}
	params.Set("key", s.apiKey)
	params.Set("part", "snippet")
	params.Set("type", "video")
	params.Set("q", query)
	params.Set("maxResults", strconv.Itoa(maxResults))
	params.Set("order", "relevance")
	if pageToken != "" {
		params.Set("pageToken", pageToken)
	}

	reqURL := fmt.Sprintf("https://www.googleapis.com/youtube/v3/search?%s", params.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

func (s *YouTubeService) GetVideoDetails(ctx context.Context, videoIDs []string) ([]VideoItem, error) {
	if len(videoIDs) == 0 {
		return nil, nil
	}

	params := url.Values{}
	params.Set("key", s.apiKey)
	params.Set("part", "snippet,contentDetails,statistics")
	params.Set("id", joinStrings(videoIDs, ","))

	reqURL := fmt.Sprintf("https://www.googleapis.com/youtube/v3/videos?%s", params.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var result VideoListResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result.Items, nil
}

func (s *YouTubeService) GetChannelVideos(ctx context.Context, channelID string, maxResults int, pageToken string) (*SearchResult, error) {
	params := url.Values{}
	params.Set("key", s.apiKey)
	params.Set("part", "snippet")
	params.Set("type", "video")
	params.Set("channelId", channelID)
	params.Set("maxResults", strconv.Itoa(maxResults))
	params.Set("order", "date")
	if pageToken != "" {
		params.Set("pageToken", pageToken)
	}

	reqURL := fmt.Sprintf("https://www.googleapis.com/youtube/v3/search?%s", params.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

func (s *YouTubeService) ToVideo(item VideoItem, taskID *uuid.UUID) *models.Video {
	video := &models.Video{
		YouTubeID:   item.ID,
		TaskID:      taskID,
		Title:       item.Snippet.Title,
		Description: &item.Snippet.Description,
		ChannelID:   &item.Snippet.ChannelID,
		ChannelName: &item.Snippet.ChannelTitle,
		Metadata:    []byte("{}"),
	}

	// Parse duration
	if duration := parseDuration(item.ContentDetails.Duration); duration > 0 {
		video.DurationSeconds = &duration
	}

	// Parse statistics
	if viewCount, err := strconv.ParseInt(item.Statistics.ViewCount, 10, 64); err == nil {
		video.ViewCount = &viewCount
	}
	if likeCount, err := strconv.ParseInt(item.Statistics.LikeCount, 10, 64); err == nil {
		video.LikeCount = &likeCount
	}
	if commentCount, err := strconv.ParseInt(item.Statistics.CommentCount, 10, 64); err == nil {
		video.CommentCount = &commentCount
	}

	// Parse publish date
	if publishedAt, err := time.Parse(time.RFC3339, item.Snippet.PublishedAt); err == nil {
		video.PublishedAt = &publishedAt
	}

	// Get best thumbnail
	if item.Snippet.Thumbnails.Maxres.URL != "" {
		video.ThumbnailURL = &item.Snippet.Thumbnails.Maxres.URL
	} else if item.Snippet.Thumbnails.High.URL != "" {
		video.ThumbnailURL = &item.Snippet.Thumbnails.High.URL
	} else if item.Snippet.Thumbnails.Medium.URL != "" {
		video.ThumbnailURL = &item.Snippet.Thumbnails.Medium.URL
	}

	return video
}

// parseDuration parses ISO 8601 duration (PT1H2M3S) to seconds
func parseDuration(duration string) int {
	re := regexp.MustCompile(`PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?`)
	matches := re.FindStringSubmatch(duration)
	if matches == nil {
		return 0
	}

	var seconds int
	if matches[1] != "" {
		h, _ := strconv.Atoi(matches[1])
		seconds += h * 3600
	}
	if matches[2] != "" {
		m, _ := strconv.Atoi(matches[2])
		seconds += m * 60
	}
	if matches[3] != "" {
		s, _ := strconv.Atoi(matches[3])
		seconds += s
	}

	return seconds
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
