package config

import (
	"os"
	"strconv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	MinIO    MinIOConfig
	YouTube  YouTubeConfig
	Claude   ClaudeConfig
	ML       MLConfig
	Worker   WorkerConfig
}

type ServerConfig struct {
	Port string
	Mode string
}

type DatabaseConfig struct {
	URL string
}

type RedisConfig struct {
	URL string
}

type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

type YouTubeConfig struct {
	APIKey string
}

type ClaudeConfig struct {
	APIKey string
}

type MLConfig struct {
	ServiceAddr string
}

type WorkerConfig struct {
	Concurrency int
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("API_PORT", "8080"),
			Mode: getEnv("GIN_MODE", "debug"),
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", "postgres://ytcrawler:ytcrawler123@localhost:5432/youtube_crawler?sslmode=disable"),
		},
		Redis: RedisConfig{
			URL: getEnv("REDIS_URL", "redis://localhost:6379"),
		},
		MinIO: MinIOConfig{
			Endpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin123"),
			Bucket:    getEnv("MINIO_BUCKET", "youtube-crawler"),
			UseSSL:    getEnvBool("MINIO_USE_SSL", false),
		},
		YouTube: YouTubeConfig{
			APIKey: getEnv("YOUTUBE_API_KEY", ""),
		},
		Claude: ClaudeConfig{
			APIKey: getEnv("ANTHROPIC_API_KEY", ""),
		},
		ML: MLConfig{
			ServiceAddr: getEnv("ML_SERVICE_ADDR", "localhost:50051"),
		},
		Worker: WorkerConfig{
			Concurrency: getEnvInt("WORKER_CONCURRENCY", 10),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}
