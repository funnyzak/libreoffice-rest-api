package repository

import "time"

// TaskStatus 任务状态。
type TaskStatus string

const (
	// TaskStatusPending 待处理。
	TaskStatusPending TaskStatus = "pending"
	// TaskStatusProcessing 处理中。
	TaskStatusProcessing TaskStatus = "processing"
	// TaskStatusSuccess 成功。
	TaskStatusSuccess TaskStatus = "success"
	// TaskStatusFailed 失败。
	TaskStatusFailed TaskStatus = "failed"
)

// Task 任务模型。
type Task struct {
	ID           string     `gorm:"primaryKey;size:36"`
	Status       TaskStatus `gorm:"size:20;index"`
	SourceType   string     `gorm:"size:20"`
	SourceName   string     `gorm:"size:255"`
	SourceURL    string     `gorm:"size:2048"`
	InputPath    string     `gorm:"size:1024"`
	OutputPath   string     `gorm:"size:1024"`
	OutputFormat string     `gorm:"size:20"`
	ErrorMessage string     `gorm:"size:1024"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ExpiresAt    time.Time `gorm:"index"`
}
