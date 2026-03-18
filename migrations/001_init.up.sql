-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Tasks table
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- 'keyword_search', 'channel_monitor', 'playlist'
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, running, paused, completed, failed
    config JSONB NOT NULL DEFAULT '{}',
    progress INTEGER DEFAULT 0,
    total_found INTEGER DEFAULT 0,
    total_analyzed INTEGER DEFAULT 0,
    total_confirmed INTEGER DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Task keywords table
CREATE TABLE task_keywords (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    keyword VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Task channels table
CREATE TABLE task_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    channel_id VARCHAR(50) NOT NULL,
    channel_name VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Videos table
CREATE TABLE videos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    youtube_id VARCHAR(20) UNIQUE NOT NULL,
    task_id UUID REFERENCES tasks(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    description TEXT,
    channel_id VARCHAR(50),
    channel_name VARCHAR(255),
    duration_seconds INTEGER,
    view_count BIGINT,
    like_count BIGINT,
    comment_count BIGINT,
    published_at TIMESTAMP WITH TIME ZONE,
    thumbnail_url TEXT,
    is_dialogue BOOLEAN,
    dialogue_confidence DECIMAL(5,4),
    face_count_avg DECIMAL(4,2),
    speaker_count INTEGER,
    analysis_status VARCHAR(50) DEFAULT 'pending', -- pending, analyzing, completed, failed
    analysis_error TEXT,
    reviewed BOOLEAN DEFAULT FALSE,
    review_result BOOLEAN,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Dialogue segments table
CREATE TABLE dialogue_segments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    start_time_ms INTEGER NOT NULL,
    end_time_ms INTEGER NOT NULL,
    speaker_count INTEGER,
    confidence DECIMAL(5,4),
    transcript TEXT,
    summary TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Video tags table
CREATE TABLE video_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    tag_name VARCHAR(100) NOT NULL,
    tag_type VARCHAR(50) NOT NULL, -- 'topic', 'format', 'guest', 'custom'
    confidence DECIMAL(5,4),
    source VARCHAR(50), -- 'auto', 'manual', 'llm'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(video_id, tag_name)
);

-- Indexes
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_type ON tasks(type);
CREATE INDEX idx_tasks_created_at ON tasks(created_at DESC);

CREATE INDEX idx_task_keywords_task_id ON task_keywords(task_id);
CREATE INDEX idx_task_channels_task_id ON task_channels(task_id);

CREATE INDEX idx_videos_task_id ON videos(task_id);
CREATE INDEX idx_videos_youtube_id ON videos(youtube_id);
CREATE INDEX idx_videos_is_dialogue ON videos(is_dialogue);
CREATE INDEX idx_videos_analysis_status ON videos(analysis_status);
CREATE INDEX idx_videos_created_at ON videos(created_at DESC);
CREATE INDEX idx_videos_channel_id ON videos(channel_id);

CREATE INDEX idx_dialogue_segments_video_id ON dialogue_segments(video_id);
CREATE INDEX idx_dialogue_segments_time ON dialogue_segments(start_time_ms, end_time_ms);

CREATE INDEX idx_video_tags_video_id ON video_tags(video_id);
CREATE INDEX idx_video_tags_tag_name ON video_tags(tag_name);
CREATE INDEX idx_video_tags_tag_type ON video_tags(tag_type);

-- Update timestamp trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_tasks_updated_at
    BEFORE UPDATE ON tasks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_videos_updated_at
    BEFORE UPDATE ON videos
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
