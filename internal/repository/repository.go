package repository

import (
	"context"
	"time"
)

// Repository 任务仓储接口。
type Repository interface {
	CreateTask(ctx context.Context, task *Task) error
	GetTask(ctx context.Context, id string) (*Task, error)
	UpdateTaskStatus(ctx context.Context, id string, status TaskStatus, errMsg string) error
	UpdateTaskResult(ctx context.Context, id string, outputPath string) error
	DeleteTask(ctx context.Context, id string) error
	ListExpiredTasks(ctx context.Context, now time.Time) ([]Task, error)
	Ping(ctx context.Context) error
}
