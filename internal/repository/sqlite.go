package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SQLiteRepository SQLite 实现。
type SQLiteRepository struct {
	db *gorm.DB
}

// NewSQLiteRepository 创建 SQLite 仓储。
func NewSQLiteRepository(path string) (*SQLiteRepository, error) {
	if path == "" {
		return nil, errors.New("数据库路径不能为空")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("创建数据库目录失败: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	if err := db.AutoMigrate(&Task{}); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	return &SQLiteRepository{db: db}, nil
}

// CreateTask 创建任务。
func (r *SQLiteRepository) CreateTask(ctx context.Context, task *Task) error {
	return r.db.WithContext(ctx).Create(task).Error
}

// GetTask 获取任务。
func (r *SQLiteRepository) GetTask(ctx context.Context, id string) (*Task, error) {
	var task Task
	if err := r.db.WithContext(ctx).First(&task, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// UpdateTaskStatus 更新任务状态。
func (r *SQLiteRepository) UpdateTaskStatus(ctx context.Context, id string, status TaskStatus, errMsg string) error {
	updates := map[string]any{
		"status":        status,
		"error_message": errMsg,
		"updated_at":    time.Now(),
	}
	return r.db.WithContext(ctx).Model(&Task{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateTaskResult 更新任务结果。
func (r *SQLiteRepository) UpdateTaskResult(ctx context.Context, id string, outputPath string) error {
	updates := map[string]any{
		"output_path": outputPath,
		"updated_at":  time.Now(),
	}
	return r.db.WithContext(ctx).Model(&Task{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteTask 删除任务。
func (r *SQLiteRepository) DeleteTask(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&Task{}, "id = ?", id).Error
}

// ListExpiredTasks 获取过期任务。
func (r *SQLiteRepository) ListExpiredTasks(ctx context.Context, now time.Time) ([]Task, error) {
	var tasks []Task
	if err := r.db.WithContext(ctx).Where("expires_at <= ?", now).Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// Ping 检测数据库连通性。
func (r *SQLiteRepository) Ping(ctx context.Context) error {
	db, err := r.db.DB()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}
