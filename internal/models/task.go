package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type TaskType string

const (
	TaskTypeKeywordSearch  TaskType = "keyword_search"
	TaskTypeChannelMonitor TaskType = "channel_monitor"
	TaskTypePlaylist       TaskType = "playlist"
)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusPaused    TaskStatus = "paused"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

type Task struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	Name           string          `json:"name" db:"name"`
	Type           TaskType        `json:"type" db:"type"`
	Status         TaskStatus      `json:"status" db:"status"`
	Config         json.RawMessage `json:"config" db:"config"`
	Progress       int             `json:"progress" db:"progress"`
	TotalFound     int             `json:"total_found" db:"total_found"`
	TotalAnalyzed  int             `json:"total_analyzed" db:"total_analyzed"`
	TotalConfirmed int             `json:"total_confirmed" db:"total_confirmed"`
	ErrorMessage   *string         `json:"error_message,omitempty" db:"error_message"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at" db:"updated_at"`
	StartedAt      *time.Time      `json:"started_at,omitempty" db:"started_at"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty" db:"completed_at"`

	// Relations (not in DB)
	Keywords []string `json:"keywords,omitempty" db:"-"`
	Channels []string `json:"channels,omitempty" db:"-"`
}

type TaskConfig struct {
	Keywords    []string `json:"keywords,omitempty"`
	ChannelIDs  []string `json:"channel_ids,omitempty"`
	PlaylistID  string   `json:"playlist_id,omitempty"`
	MaxResults  int      `json:"max_results,omitempty"`
	DateAfter   string   `json:"date_after,omitempty"`
	DateBefore  string   `json:"date_before,omitempty"`
	MinDuration int      `json:"min_duration,omitempty"` // seconds
	MaxDuration int      `json:"max_duration,omitempty"` // seconds
	Language    string   `json:"language,omitempty"`
}

type TaskKeyword struct {
	ID        uuid.UUID `json:"id" db:"id"`
	TaskID    uuid.UUID `json:"task_id" db:"task_id"`
	Keyword   string    `json:"keyword" db:"keyword"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type TaskChannel struct {
	ID          uuid.UUID `json:"id" db:"id"`
	TaskID      uuid.UUID `json:"task_id" db:"task_id"`
	ChannelID   string    `json:"channel_id" db:"channel_id"`
	ChannelName string    `json:"channel_name" db:"channel_name"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type CreateTaskRequest struct {
	Name   string     `json:"name" binding:"required"`
	Type   TaskType   `json:"type" binding:"required,oneof=keyword_search channel_monitor playlist"`
	Config TaskConfig `json:"config" binding:"required"`
}

type UpdateTaskRequest struct {
	Name   *string     `json:"name,omitempty"`
	Config *TaskConfig `json:"config,omitempty"`
}

type TaskListParams struct {
	Page     int        `form:"page,default=1"`
	PageSize int        `form:"page_size,default=20"`
	Status   TaskStatus `form:"status"`
	Type     TaskType   `form:"type"`
	Search   string     `form:"search"`
}
