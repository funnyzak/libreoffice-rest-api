package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSQLiteRepositoryCRUD(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("创建仓储失败: %v", err)
	}

	ctx := context.Background()
	task := &Task{
		ID:           "task-1",
		Status:       TaskStatusPending,
		SourceType:   "upload",
		SourceName:   "demo.docx",
		InputPath:    "/tmp/demo.docx",
		OutputFormat: "pdf",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(2 * time.Hour),
	}

	if err := repo.CreateTask(ctx, task); err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}

	stored, err := repo.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("查询任务失败: %v", err)
	}
	if stored.ID != task.ID {
		t.Fatalf("任务 ID 不匹配")
	}

	if err := repo.UpdateTaskStatus(ctx, task.ID, TaskStatusProcessing, ""); err != nil {
		t.Fatalf("更新状态失败: %v", err)
	}

	if err := repo.UpdateTaskResult(ctx, task.ID, "/tmp/out.pdf"); err != nil {
		t.Fatalf("更新结果失败: %v", err)
	}

	stored, _ = repo.GetTask(ctx, task.ID)
	if stored.OutputPath == "" {
		t.Fatalf("输出路径未更新")
	}

	if err := repo.DeleteTask(ctx, task.ID); err != nil {
		t.Fatalf("删除任务失败: %v", err)
	}

	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("数据库文件不存在: %v", err)
	}
}

func TestSQLiteRepositoryListExpired(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test-expired.db")
	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("创建仓储失败: %v", err)
	}

	ctx := context.Background()
	_ = repo.CreateTask(ctx, &Task{
		ID:        "expired",
		Status:    TaskStatusSuccess,
		ExpiresAt: time.Now().Add(-time.Hour),
	})

	_ = repo.CreateTask(ctx, &Task{
		ID:        "active",
		Status:    TaskStatusSuccess,
		ExpiresAt: time.Now().Add(time.Hour),
	})

	items, err := repo.ListExpiredTasks(ctx, time.Now())
	if err != nil {
		t.Fatalf("查询过期任务失败: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("期望仅有 1 条过期任务")
	}
}
