package models

import (
	"time"

	"github.com/google/uuid"
)

type TagType string

const (
	TagTypeTopic  TagType = "topic"
	TagTypeFormat TagType = "format"
	TagTypeGuest  TagType = "guest"
	TagTypeCustom TagType = "custom"
)

type TagSource string

const (
	TagSourceAuto   TagSource = "auto"
	TagSourceManual TagSource = "manual"
	TagSourceLLM    TagSource = "llm"
)

type VideoTag struct {
	ID         uuid.UUID `json:"id" db:"id"`
	VideoID    uuid.UUID `json:"video_id" db:"video_id"`
	TagName    string    `json:"tag_name" db:"tag_name"`
	TagType    TagType   `json:"tag_type" db:"tag_type"`
	Confidence *float64  `json:"confidence,omitempty" db:"confidence"`
	Source     TagSource `json:"source" db:"source"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type CreateTagRequest struct {
	TagName    string   `json:"tag_name" binding:"required,max=100"`
	TagType    TagType  `json:"tag_type" binding:"required,oneof=topic format guest custom"`
	Confidence *float64 `json:"confidence,omitempty"`
}

type TagStats struct {
	TagName string `json:"tag_name"`
	TagType TagType `json:"tag_type"`
	Count   int    `json:"count"`
}
