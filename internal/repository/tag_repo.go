package repository

import (
	"context"
	"fmt"

	"github.com/example/youtube-dialogue-crawler/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TagRepository struct {
	db *pgxpool.Pool
}

func NewTagRepository(db *pgxpool.Pool) *TagRepository {
	return &TagRepository{db: db}
}

func (r *TagRepository) Create(ctx context.Context, videoID uuid.UUID, req *models.CreateTagRequest) (*models.VideoTag, error) {
	tag := &models.VideoTag{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO video_tags (video_id, tag_name, tag_type, confidence, source)
		VALUES ($1, $2, $3, $4, 'manual')
		ON CONFLICT (video_id, tag_name) DO UPDATE SET confidence = $4
		RETURNING id, video_id, tag_name, tag_type, confidence, source, created_at
	`, videoID, req.TagName, req.TagType, req.Confidence).Scan(
		&tag.ID, &tag.VideoID, &tag.TagName, &tag.TagType, &tag.Confidence, &tag.Source, &tag.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert tag: %w", err)
	}
	return tag, nil
}

func (r *TagRepository) CreateBatch(ctx context.Context, videoID uuid.UUID, tags []models.VideoTag) error {
	for _, tag := range tags {
		_, err := r.db.Exec(ctx, `
			INSERT INTO video_tags (video_id, tag_name, tag_type, confidence, source)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (video_id, tag_name) DO NOTHING
		`, videoID, tag.TagName, tag.TagType, tag.Confidence, tag.Source)
		if err != nil {
			return fmt.Errorf("insert tag %s: %w", tag.TagName, err)
		}
	}
	return nil
}

func (r *TagRepository) GetByVideoID(ctx context.Context, videoID uuid.UUID) ([]models.VideoTag, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, video_id, tag_name, tag_type, confidence, source, created_at
		FROM video_tags
		WHERE video_id = $1
		ORDER BY created_at DESC
	`, videoID)
	if err != nil {
		return nil, fmt.Errorf("query tags: %w", err)
	}
	defer rows.Close()

	var tags []models.VideoTag
	for rows.Next() {
		var tag models.VideoTag
		err := rows.Scan(&tag.ID, &tag.VideoID, &tag.TagName, &tag.TagType, &tag.Confidence, &tag.Source, &tag.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	return tags, nil
}

func (r *TagRepository) Delete(ctx context.Context, tagID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM video_tags WHERE id = $1`, tagID)
	return err
}

func (r *TagRepository) GetStats(ctx context.Context) ([]models.TagStats, error) {
	rows, err := r.db.Query(ctx, `
		SELECT tag_name, tag_type, COUNT(*) as count
		FROM video_tags
		GROUP BY tag_name, tag_type
		ORDER BY count DESC
		LIMIT 100
	`)
	if err != nil {
		return nil, fmt.Errorf("query tag stats: %w", err)
	}
	defer rows.Close()

	var stats []models.TagStats
	for rows.Next() {
		var s models.TagStats
		err := rows.Scan(&s.TagName, &s.TagType, &s.Count)
		if err != nil {
			return nil, fmt.Errorf("scan stats: %w", err)
		}
		stats = append(stats, s)
	}

	return stats, nil
}

// Segment repository functions
func (r *TagRepository) CreateSegment(ctx context.Context, videoID uuid.UUID, req *models.CreateSegmentRequest) (*models.DialogueSegment, error) {
	segment := &models.DialogueSegment{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO dialogue_segments (video_id, start_time_ms, end_time_ms, speaker_count, confidence, transcript, summary)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, video_id, start_time_ms, end_time_ms, speaker_count, confidence, transcript, summary, created_at
	`, videoID, req.StartTimeMs, req.EndTimeMs, req.SpeakerCount, req.Confidence, req.Transcript, req.Summary).Scan(
		&segment.ID, &segment.VideoID, &segment.StartTimeMs, &segment.EndTimeMs,
		&segment.SpeakerCount, &segment.Confidence, &segment.Transcript, &segment.Summary, &segment.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert segment: %w", err)
	}
	return segment, nil
}

func (r *TagRepository) CreateSegmentBatch(ctx context.Context, videoID uuid.UUID, segments []models.DialogueSegment) error {
	for _, seg := range segments {
		_, err := r.db.Exec(ctx, `
			INSERT INTO dialogue_segments (video_id, start_time_ms, end_time_ms, speaker_count, confidence, transcript, summary)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, videoID, seg.StartTimeMs, seg.EndTimeMs, seg.SpeakerCount, seg.Confidence, seg.Transcript, seg.Summary)
		if err != nil {
			return fmt.Errorf("insert segment: %w", err)
		}
	}
	return nil
}

func (r *TagRepository) GetSegmentsByVideoID(ctx context.Context, videoID uuid.UUID) ([]models.DialogueSegment, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, video_id, start_time_ms, end_time_ms, speaker_count, confidence, transcript, summary, created_at
		FROM dialogue_segments
		WHERE video_id = $1
		ORDER BY start_time_ms ASC
	`, videoID)
	if err != nil {
		return nil, fmt.Errorf("query segments: %w", err)
	}
	defer rows.Close()

	var segments []models.DialogueSegment
	for rows.Next() {
		var seg models.DialogueSegment
		err := rows.Scan(&seg.ID, &seg.VideoID, &seg.StartTimeMs, &seg.EndTimeMs,
			&seg.SpeakerCount, &seg.Confidence, &seg.Transcript, &seg.Summary, &seg.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan segment: %w", err)
		}
		segments = append(segments, seg)
	}

	return segments, nil
}

func (r *TagRepository) DeleteSegmentsByVideoID(ctx context.Context, videoID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM dialogue_segments WHERE video_id = $1`, videoID)
	return err
}
