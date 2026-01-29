package service

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/funnyzak/libreoffice-rest-api/internal/repository"
	domainerrors "github.com/funnyzak/libreoffice-rest-api/pkg/errors"
	"gorm.io/gorm"
)

// TaskService 任务服务。
type TaskService struct {
	repo   repository.Repository
	logger zerolog.Logger
}

// NewTaskService 创建任务服务。
func NewTaskService(repo repository.Repository, logger zerolog.Logger) *TaskService {
	return &TaskService{
		repo:   repo,
		logger: logger,
	}
}

// GetTask 获取任务。
func (s *TaskService) GetTask(ctx context.Context, id string) (*repository.Task, error) {
	s.logger.Debug().Str("task_id", id).Msg("读取任务信息")
	task, err := s.repo.GetTask(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			s.logger.Warn().Str("task_id", id).Msg("任务不存在")
			return nil, domainerrors.NewNotFound("任务不存在", "任务 ID 不存在", err)
		}
		s.logger.Error().Err(err).Str("task_id", id).Msg("获取任务失败")
		return nil, domainerrors.NewStorage("获取任务失败", "数据库读取失败", err)
	}
	s.logger.Debug().
		Str("task_id", id).
		Str("status", string(task.Status)).
		Msg("任务读取成功")
	return task, nil
}
