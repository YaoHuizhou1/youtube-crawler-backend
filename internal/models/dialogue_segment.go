package models

import (
	"time"

	"github.com/google/uuid"
)

type DialogueSegment struct {
	ID           uuid.UUID `json:"id" db:"id"`
	VideoID      uuid.UUID `json:"video_id" db:"video_id"`
	StartTimeMs  int       `json:"start_time_ms" db:"start_time_ms"`
	EndTimeMs    int       `json:"end_time_ms" db:"end_time_ms"`
	SpeakerCount *int      `json:"speaker_count,omitempty" db:"speaker_count"`
	Confidence   *float64  `json:"confidence,omitempty" db:"confidence"`
	Transcript   *string   `json:"transcript,omitempty" db:"transcript"`
	Summary      *string   `json:"summary,omitempty" db:"summary"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type CreateSegmentRequest struct {
	StartTimeMs  int      `json:"start_time_ms" binding:"required,gte=0"`
	EndTimeMs    int      `json:"end_time_ms" binding:"required,gtfield=StartTimeMs"`
	SpeakerCount *int     `json:"speaker_count,omitempty"`
	Confidence   *float64 `json:"confidence,omitempty"`
	Transcript   *string  `json:"transcript,omitempty"`
	Summary      *string  `json:"summary,omitempty"`
}
