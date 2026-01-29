package cleaner

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"

	"github.com/funnyzak/libreoffice-rest-api/internal/repository"
)

// Cleaner 文件清理器。
type Cleaner struct {
	repo     repository.Repository
	interval time.Duration
	logger   zerolog.Logger
}

// New 创建清理器。
func New(repo repository.Repository, interval time.Duration, logger zerolog.Logger) *Cleaner {
	return &Cleaner{
		repo:     repo,
		interval: interval,
		logger:   logger,
	}
}

// Start 启动清理任务。
func (c *Cleaner) Start(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	c.logger.Info().
		Dur("interval", c.interval).
		Msg("清理器启动")

	for {
		select {
		case <-ctx.Done():
			c.logger.Info().Msg("清理器停止")
			return
		case <-ticker.C:
			c.cleanup(ctx)
		}
	}
}

func (c *Cleaner) cleanup(ctx context.Context) {
	tasks, err := c.repo.ListExpiredTasks(ctx, time.Now())
	if err != nil {
		c.logger.Error().Err(err).Msg("获取过期任务失败")
		return
	}
	c.logger.Debug().Int("count", len(tasks)).Msg("开始清理过期任务")

	for _, task := range tasks {
		if task.OutputPath != "" {
			_ = os.Remove(task.OutputPath)
			_ = os.RemoveAll(filepath.Dir(task.OutputPath))
		}
		if err := c.repo.DeleteTask(ctx, task.ID); err != nil {
			c.logger.Error().Err(err).Str("task_id", task.ID).Msg("删除任务记录失败")
		} else {
			c.logger.Info().Str("task_id", task.ID).Msg("过期任务已删除")
		}
	}
}
