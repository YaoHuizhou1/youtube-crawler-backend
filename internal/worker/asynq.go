package worker

import (
	"github.com/hibiken/asynq"
)

const (
	TaskTypeDiscovery = "discovery:run"
	TaskTypeAnalysis  = "analysis:analyze"
	TaskTypeTagging   = "tagging:tag"
)

func NewServer(redisAddr string, concurrency int) *asynq.Server {
	return asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Concurrency: concurrency,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)
}

func NewClient(redisAddr string) *asynq.Client {
	return asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
}

func NewScheduler(redisAddr string) *asynq.Scheduler {
	return asynq.NewScheduler(
		asynq.RedisClientOpt{Addr: redisAddr},
		nil,
	)
}
