-- Drop triggers
DROP TRIGGER IF EXISTS update_videos_updated_at ON videos;
DROP TRIGGER IF EXISTS update_tasks_updated_at ON tasks;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order of creation
DROP TABLE IF EXISTS video_tags;
DROP TABLE IF EXISTS dialogue_segments;
DROP TABLE IF EXISTS videos;
DROP TABLE IF EXISTS task_channels;
DROP TABLE IF EXISTS task_keywords;
DROP TABLE IF EXISTS tasks;
