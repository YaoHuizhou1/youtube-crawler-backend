package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AnalysisStatus string

const (
	AnalysisStatusPending   AnalysisStatus = "pending"
	AnalysisStatusAnalyzing AnalysisStatus = "analyzing"
	AnalysisStatusCompleted AnalysisStatus = "completed"
	AnalysisStatusFailed    AnalysisStatus = "failed"
)

type Video struct {
	ID                 uuid.UUID       `json:"id" db:"id"`
	YouTubeID          string          `json:"youtube_id" db:"youtube_id"`
	TaskID             *uuid.UUID      `json:"task_id,omitempty" db:"task_id"`
	Title              string          `json:"title" db:"title"`
	Description        *string         `json:"description,omitempty" db:"description"`
	ChannelID          *string         `json:"channel_id,omitempty" db:"channel_id"`
	ChannelName        *string         `json:"channel_name,omitempty" db:"channel_name"`
	DurationSeconds    *int            `json:"duration_seconds,omitempty" db:"duration_seconds"`
	ViewCount          *int64          `json:"view_count,omitempty" db:"view_count"`
	LikeCount          *int64          `json:"like_count,omitempty" db:"like_count"`
	CommentCount       *int64          `json:"comment_count,omitempty" db:"comment_count"`
	PublishedAt        *time.Time      `json:"published_at,omitempty" db:"published_at"`
	ThumbnailURL       *string         `json:"thumbnail_url,omitempty" db:"thumbnail_url"`
	IsDialogue         *bool           `json:"is_dialogue,omitempty" db:"is_dialogue"`
	DialogueConfidence *float64        `json:"dialogue_confidence,omitempty" db:"dialogue_confidence"`
	FaceCountAvg       *float64        `json:"face_count_avg,omitempty" db:"face_count_avg"`
	SpeakerCount       *int            `json:"speaker_count,omitempty" db:"speaker_count"`
	AnalysisStatus     AnalysisStatus  `json:"analysis_status" db:"analysis_status"`
	AnalysisError      *string         `json:"analysis_error,omitempty" db:"analysis_error"`
	Reviewed           bool            `json:"reviewed" db:"reviewed"`
	ReviewResult       *bool           `json:"review_result,omitempty" db:"review_result"`
	Metadata           json.RawMessage `json:"metadata" db:"metadata"`
	CreatedAt          time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at" db:"updated_at"`

	// Relations (not in DB)
	Tags     []VideoTag         `json:"tags,omitempty" db:"-"`
	Segments []DialogueSegment  `json:"segments,omitempty" db:"-"`
}

type VideoListParams struct {
	Page           int            `form:"page,default=1"`
	PageSize       int            `form:"page_size,default=20"`
	TaskID         *uuid.UUID     `form:"task_id"`
	IsDialogue     *bool          `form:"is_dialogue"`
	AnalysisStatus AnalysisStatus `form:"analysis_status"`
	Reviewed       *bool          `form:"reviewed"`
	Tags           []string       `form:"tags"`
	Search         string         `form:"search"`
	SortBy         string         `form:"sort_by,default=created_at"`
	SortOrder      string         `form:"sort_order,default=desc"`
}

type ReviewVideoRequest struct {
	Result bool `json:"result" binding:"required"`
}

type VideoMetadata struct {
	VisualScore  float64 `json:"visual_score,omitempty"`
	AudioScore   float64 `json:"audio_score,omitempty"`
	MetaScore    float64 `json:"meta_score,omitempty"`
	AnalysisTime int64   `json:"analysis_time_ms,omitempty"`
}

type ExportRequest struct {
	Format     string     `json:"format" binding:"required,oneof=csv json"`
	TaskID     *uuid.UUID `json:"task_id,omitempty"`
	IsDialogue *bool      `json:"is_dialogue,omitempty"`
	Tags       []string   `json:"tags,omitempty"`
}
