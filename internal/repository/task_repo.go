package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/example/youtube-dialogue-crawler/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TaskRepository struct {
	db *pgxpool.Pool
}

func NewTaskRepository(db *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(ctx context.Context, req *models.CreateTaskRequest) (*models.Task, error) {
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}

	task := &models.Task{}
	err = r.db.QueryRow(ctx, `
		INSERT INTO tasks (name, type, config)
		VALUES ($1, $2, $3)
		RETURNING id, name, type, status, config, progress, total_found, total_analyzed,
		          total_confirmed, error_message, created_at, updated_at, started_at, completed_at
	`, req.Name, req.Type, configJSON).Scan(
		&task.ID, &task.Name, &task.Type, &task.Status, &task.Config, &task.Progress,
		&task.TotalFound, &task.TotalAnalyzed, &task.TotalConfirmed, &task.ErrorMessage,
		&task.CreatedAt, &task.UpdatedAt, &task.StartedAt, &task.CompletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert task: %w", err)
	}

	// Insert keywords
	if len(req.Config.Keywords) > 0 {
		for _, keyword := range req.Config.Keywords {
			_, err = r.db.Exec(ctx, `
				INSERT INTO task_keywords (task_id, keyword) VALUES ($1, $2)
			`, task.ID, keyword)
			if err != nil {
				return nil, fmt.Errorf("insert keyword: %w", err)
			}
		}
		task.Keywords = req.Config.Keywords
	}

	// Insert channels
	if len(req.Config.ChannelIDs) > 0 {
		for _, channelID := range req.Config.ChannelIDs {
			_, err = r.db.Exec(ctx, `
				INSERT INTO task_channels (task_id, channel_id) VALUES ($1, $2)
			`, task.ID, channelID)
			if err != nil {
				return nil, fmt.Errorf("insert channel: %w", err)
			}
		}
		task.Channels = req.Config.ChannelIDs
	}

	return task, nil
}

func (r *TaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Task, error) {
	task := &models.Task{}
	err := r.db.QueryRow(ctx, `
		SELECT id, name, type, status, config, progress, total_found, total_analyzed,
		       total_confirmed, error_message, created_at, updated_at, started_at, completed_at
		FROM tasks WHERE id = $1
	`, id).Scan(
		&task.ID, &task.Name, &task.Type, &task.Status, &task.Config, &task.Progress,
		&task.TotalFound, &task.TotalAnalyzed, &task.TotalConfirmed, &task.ErrorMessage,
		&task.CreatedAt, &task.UpdatedAt, &task.StartedAt, &task.CompletedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query task: %w", err)
	}

	// Load keywords
	rows, err := r.db.Query(ctx, `SELECT keyword FROM task_keywords WHERE task_id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("query keywords: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var keyword string
		if err := rows.Scan(&keyword); err != nil {
			return nil, fmt.Errorf("scan keyword: %w", err)
		}
		task.Keywords = append(task.Keywords, keyword)
	}

	// Load channels
	rows, err = r.db.Query(ctx, `SELECT channel_id FROM task_channels WHERE task_id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("query channels: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var channelID string
		if err := rows.Scan(&channelID); err != nil {
			return nil, fmt.Errorf("scan channel: %w", err)
		}
		task.Channels = append(task.Channels, channelID)
	}

	return task, nil
}

func (r *TaskRepository) List(ctx context.Context, params *models.TaskListParams) ([]*models.Task, int64, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	if params.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, params.Status)
		argNum++
	}

	if params.Type != "" {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argNum))
		args = append(args, params.Type)
		argNum++
	}

	if params.Search != "" {
		conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", argNum))
		args = append(args, "%"+params.Search+"%")
		argNum++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tasks %s", whereClause)
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count tasks: %w", err)
	}

	// Query tasks
	offset := (params.Page - 1) * params.PageSize
	query := fmt.Sprintf(`
		SELECT id, name, type, status, config, progress, total_found, total_analyzed,
		       total_confirmed, error_message, created_at, updated_at, started_at, completed_at
		FROM tasks %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, params.PageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		task := &models.Task{}
		err := rows.Scan(
			&task.ID, &task.Name, &task.Type, &task.Status, &task.Config, &task.Progress,
			&task.TotalFound, &task.TotalAnalyzed, &task.TotalConfirmed, &task.ErrorMessage,
			&task.CreatedAt, &task.UpdatedAt, &task.StartedAt, &task.CompletedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	return tasks, total, nil
}

func (r *TaskRepository) Update(ctx context.Context, id uuid.UUID, req *models.UpdateTaskRequest) (*models.Task, error) {
	var sets []string
	var args []interface{}
	argNum := 1

	if req.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argNum))
		args = append(args, *req.Name)
		argNum++
	}

	if req.Config != nil {
		configJSON, err := json.Marshal(req.Config)
		if err != nil {
			return nil, fmt.Errorf("marshal config: %w", err)
		}
		sets = append(sets, fmt.Sprintf("config = $%d", argNum))
		args = append(args, configJSON)
		argNum++
	}

	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}

	args = append(args, id)
	query := fmt.Sprintf(`
		UPDATE tasks SET %s WHERE id = $%d
	`, strings.Join(sets, ", "), argNum)

	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	return r.GetByID(ctx, id)
}

func (r *TaskRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.TaskStatus, errMsg *string) error {
	var query string
	var args []interface{}

	switch status {
	case models.TaskStatusRunning:
		query = `UPDATE tasks SET status = $1, started_at = $2 WHERE id = $3`
		args = []interface{}{status, time.Now(), id}
	case models.TaskStatusCompleted, models.TaskStatusFailed:
		query = `UPDATE tasks SET status = $1, completed_at = $2, error_message = $3 WHERE id = $4`
		args = []interface{}{status, time.Now(), errMsg, id}
	default:
		query = `UPDATE tasks SET status = $1 WHERE id = $2`
		args = []interface{}{status, id}
	}

	_, err := r.db.Exec(ctx, query, args...)
	return err
}

func (r *TaskRepository) UpdateProgress(ctx context.Context, id uuid.UUID, progress, found, analyzed, confirmed int) error {
	_, err := r.db.Exec(ctx, `
		UPDATE tasks SET progress = $1, total_found = $2, total_analyzed = $3, total_confirmed = $4
		WHERE id = $5
	`, progress, found, analyzed, confirmed, id)
	return err
}

func (r *TaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	return err
}
