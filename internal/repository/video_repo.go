package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/example/youtube-dialogue-crawler/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VideoRepository struct {
	db *pgxpool.Pool
}

func NewVideoRepository(db *pgxpool.Pool) *VideoRepository {
	return &VideoRepository{db: db}
}

func (r *VideoRepository) Create(ctx context.Context, video *models.Video) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO videos (youtube_id, task_id, title, description, channel_id, channel_name,
		                    duration_seconds, view_count, like_count, comment_count, published_at,
		                    thumbnail_url, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (youtube_id) DO NOTHING
	`, video.YouTubeID, video.TaskID, video.Title, video.Description, video.ChannelID,
		video.ChannelName, video.DurationSeconds, video.ViewCount, video.LikeCount,
		video.CommentCount, video.PublishedAt, video.ThumbnailURL, video.Metadata)
	return err
}

func (r *VideoRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Video, error) {
	video := &models.Video{}
	err := r.db.QueryRow(ctx, `
		SELECT id, youtube_id, task_id, title, description, channel_id, channel_name,
		       duration_seconds, view_count, like_count, comment_count, published_at,
		       thumbnail_url, is_dialogue, dialogue_confidence, face_count_avg, speaker_count,
		       analysis_status, analysis_error, reviewed, review_result, metadata, created_at, updated_at
		FROM videos WHERE id = $1
	`, id).Scan(
		&video.ID, &video.YouTubeID, &video.TaskID, &video.Title, &video.Description,
		&video.ChannelID, &video.ChannelName, &video.DurationSeconds, &video.ViewCount,
		&video.LikeCount, &video.CommentCount, &video.PublishedAt, &video.ThumbnailURL,
		&video.IsDialogue, &video.DialogueConfidence, &video.FaceCountAvg, &video.SpeakerCount,
		&video.AnalysisStatus, &video.AnalysisError, &video.Reviewed, &video.ReviewResult,
		&video.Metadata, &video.CreatedAt, &video.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query video: %w", err)
	}
	return video, nil
}

func (r *VideoRepository) GetByYouTubeID(ctx context.Context, youtubeID string) (*models.Video, error) {
	video := &models.Video{}
	err := r.db.QueryRow(ctx, `
		SELECT id, youtube_id, task_id, title, description, channel_id, channel_name,
		       duration_seconds, view_count, like_count, comment_count, published_at,
		       thumbnail_url, is_dialogue, dialogue_confidence, face_count_avg, speaker_count,
		       analysis_status, analysis_error, reviewed, review_result, metadata, created_at, updated_at
		FROM videos WHERE youtube_id = $1
	`, youtubeID).Scan(
		&video.ID, &video.YouTubeID, &video.TaskID, &video.Title, &video.Description,
		&video.ChannelID, &video.ChannelName, &video.DurationSeconds, &video.ViewCount,
		&video.LikeCount, &video.CommentCount, &video.PublishedAt, &video.ThumbnailURL,
		&video.IsDialogue, &video.DialogueConfidence, &video.FaceCountAvg, &video.SpeakerCount,
		&video.AnalysisStatus, &video.AnalysisError, &video.Reviewed, &video.ReviewResult,
		&video.Metadata, &video.CreatedAt, &video.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query video: %w", err)
	}
	return video, nil
}

func (r *VideoRepository) List(ctx context.Context, params *models.VideoListParams) ([]*models.Video, int64, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	if params.TaskID != nil {
		conditions = append(conditions, fmt.Sprintf("task_id = $%d", argNum))
		args = append(args, *params.TaskID)
		argNum++
	}

	if params.IsDialogue != nil {
		conditions = append(conditions, fmt.Sprintf("is_dialogue = $%d", argNum))
		args = append(args, *params.IsDialogue)
		argNum++
	}

	if params.AnalysisStatus != "" {
		conditions = append(conditions, fmt.Sprintf("analysis_status = $%d", argNum))
		args = append(args, params.AnalysisStatus)
		argNum++
	}

	if params.Reviewed != nil {
		conditions = append(conditions, fmt.Sprintf("reviewed = $%d", argNum))
		args = append(args, *params.Reviewed)
		argNum++
	}

	if params.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d)", argNum, argNum))
		args = append(args, "%"+params.Search+"%")
		argNum++
	}

	if len(params.Tags) > 0 {
		conditions = append(conditions, fmt.Sprintf(`
			id IN (SELECT video_id FROM video_tags WHERE tag_name = ANY($%d))
		`, argNum))
		args = append(args, params.Tags)
		argNum++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM videos %s", whereClause)
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count videos: %w", err)
	}

	// Validate sort fields
	sortBy := "created_at"
	if params.SortBy != "" {
		validSorts := map[string]bool{
			"created_at": true, "published_at": true, "view_count": true,
			"like_count": true, "dialogue_confidence": true,
		}
		if validSorts[params.SortBy] {
			sortBy = params.SortBy
		}
	}
	sortOrder := "DESC"
	if params.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	// Query videos
	offset := (params.Page - 1) * params.PageSize
	query := fmt.Sprintf(`
		SELECT id, youtube_id, task_id, title, description, channel_id, channel_name,
		       duration_seconds, view_count, like_count, comment_count, published_at,
		       thumbnail_url, is_dialogue, dialogue_confidence, face_count_avg, speaker_count,
		       analysis_status, analysis_error, reviewed, review_result, metadata, created_at, updated_at
		FROM videos %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, sortOrder, argNum, argNum+1)
	args = append(args, params.PageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query videos: %w", err)
	}
	defer rows.Close()

	var videos []*models.Video
	for rows.Next() {
		video := &models.Video{}
		err := rows.Scan(
			&video.ID, &video.YouTubeID, &video.TaskID, &video.Title, &video.Description,
			&video.ChannelID, &video.ChannelName, &video.DurationSeconds, &video.ViewCount,
			&video.LikeCount, &video.CommentCount, &video.PublishedAt, &video.ThumbnailURL,
			&video.IsDialogue, &video.DialogueConfidence, &video.FaceCountAvg, &video.SpeakerCount,
			&video.AnalysisStatus, &video.AnalysisError, &video.Reviewed, &video.ReviewResult,
			&video.Metadata, &video.CreatedAt, &video.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan video: %w", err)
		}
		videos = append(videos, video)
	}

	return videos, total, nil
}

func (r *VideoRepository) UpdateAnalysis(ctx context.Context, id uuid.UUID, isDialogue bool, confidence float64, faceCountAvg float64, speakerCount int, metadata []byte) error {
	_, err := r.db.Exec(ctx, `
		UPDATE videos SET
			is_dialogue = $1,
			dialogue_confidence = $2,
			face_count_avg = $3,
			speaker_count = $4,
			metadata = $5,
			analysis_status = 'completed'
		WHERE id = $6
	`, isDialogue, confidence, faceCountAvg, speakerCount, metadata, id)
	return err
}

func (r *VideoRepository) UpdateAnalysisStatus(ctx context.Context, id uuid.UUID, status models.AnalysisStatus, errMsg *string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE videos SET analysis_status = $1, analysis_error = $2 WHERE id = $3
	`, status, errMsg, id)
	return err
}

func (r *VideoRepository) UpdateReview(ctx context.Context, id uuid.UUID, result bool) error {
	_, err := r.db.Exec(ctx, `
		UPDATE videos SET reviewed = true, review_result = $1 WHERE id = $2
	`, result, id)
	return err
}

func (r *VideoRepository) GetPendingAnalysis(ctx context.Context, limit int) ([]*models.Video, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, youtube_id, task_id, title, description, channel_id, channel_name,
		       duration_seconds, view_count, like_count, comment_count, published_at,
		       thumbnail_url, is_dialogue, dialogue_confidence, face_count_avg, speaker_count,
		       analysis_status, analysis_error, reviewed, review_result, metadata, created_at, updated_at
		FROM videos
		WHERE analysis_status = 'pending'
		ORDER BY created_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query pending videos: %w", err)
	}
	defer rows.Close()

	var videos []*models.Video
	for rows.Next() {
		video := &models.Video{}
		err := rows.Scan(
			&video.ID, &video.YouTubeID, &video.TaskID, &video.Title, &video.Description,
			&video.ChannelID, &video.ChannelName, &video.DurationSeconds, &video.ViewCount,
			&video.LikeCount, &video.CommentCount, &video.PublishedAt, &video.ThumbnailURL,
			&video.IsDialogue, &video.DialogueConfidence, &video.FaceCountAvg, &video.SpeakerCount,
			&video.AnalysisStatus, &video.AnalysisError, &video.Reviewed, &video.ReviewResult,
			&video.Metadata, &video.CreatedAt, &video.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan video: %w", err)
		}
		videos = append(videos, video)
	}

	return videos, nil
}

func (r *VideoRepository) Exists(ctx context.Context, youtubeID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM videos WHERE youtube_id = $1)`, youtubeID).Scan(&exists)
	return exists, err
}
