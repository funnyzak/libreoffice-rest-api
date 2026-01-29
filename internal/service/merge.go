package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pdfcpu/pdfcpu/pkg/api"

	"github.com/funnyzak/libreoffice-rest-api/internal/repository"
	domainerrors "github.com/funnyzak/libreoffice-rest-api/pkg/errors"
)

const mergeOutputFormat = "pdf"

// MergeSync 同步合并文件。
func (s *ConverterService) MergeSync(ctx context.Context, sources []Source, format string) (*ConvertResult, error) {
	normalized, err := normalizeMergeFormat(format)
	if err != nil {
		return nil, err
	}
	if err := validateMergeSources(sources); err != nil {
		return nil, err
	}

	prepared, cleanup, err := s.prepareMergeSources(ctx, sources)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	outputDir := filepath.Join(s.cfg.Storage.OutputDir, uuid.New().String())
	outputPath, err := s.mergeToPDF(ctx, prepared, outputDir)
	if err != nil {
		_ = os.RemoveAll(outputDir)
		return nil, err
	}

	result := buildResult("", outputPath)
	result.OutputExt = normalized
	s.logger.Info().
		Str("output_path", outputPath).
		Msg("同步合并完成")
	return result, nil
}

// MergeAsync 异步合并文件。
func (s *ConverterService) MergeAsync(ctx context.Context, sources []Source, format string, baseURL string) (*ConvertResult, error) {
	normalized, err := normalizeMergeFormat(format)
	if err != nil {
		return nil, err
	}
	if err := validateMergeSources(sources); err != nil {
		return nil, err
	}

	prepared, cleanup, err := s.prepareMergeSources(ctx, sources)
	if err != nil {
		return nil, err
	}

	taskID := uuid.New().String()
	expiresAt := time.Now().Add(time.Duration(s.cfg.Storage.RetentionHours) * time.Hour)
	task := &repository.Task{
		ID:           taskID,
		Status:       repository.TaskStatusPending,
		SourceType:   "merge",
		SourceName:   buildMergeSourceName(prepared),
		OutputFormat: normalized,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		ExpiresAt:    expiresAt,
	}

	if err := s.repo.CreateTask(ctx, task); err != nil {
		cleanup()
		return nil, domainerrors.NewStorage("创建任务失败", "数据库写入失败", err)
	}
	if s.metrics != nil {
		s.metrics.TaskTotal.WithLabelValues("pending").Inc()
	}
	s.logger.Info().
		Str("task_id", taskID).
		Int("source_count", len(prepared)).
		Time("expires_at", task.ExpiresAt).
		Msg("合并任务已创建")

	job := func(jobCtx context.Context) error {
		defer cleanup()
		if s.metrics != nil {
			s.metrics.ActiveTasks.Inc()
			defer s.metrics.ActiveTasks.Dec()
		}
		s.logger.Debug().
			Str("task_id", taskID).
			Msg("开始执行合并任务")
		if err := s.repo.UpdateTaskStatus(jobCtx, taskID, repository.TaskStatusProcessing, ""); err != nil {
			s.logger.Error().Err(err).Msg("更新任务状态失败")
		}

		outputDir := filepath.Join(s.cfg.Storage.OutputDir, taskID)
		outputPath, err := s.mergeToPDF(jobCtx, prepared, outputDir)
		if err != nil {
			_ = os.RemoveAll(outputDir)
			s.logger.Error().Err(err).Str("task_id", taskID).Msg("合并执行失败")
			_ = s.repo.UpdateTaskStatus(jobCtx, taskID, repository.TaskStatusFailed, err.Error())
			if s.metrics != nil {
				s.metrics.TaskTotal.WithLabelValues("failed").Inc()
			}
			return err
		}

		if err := s.repo.UpdateTaskResult(jobCtx, taskID, outputPath); err != nil {
			s.logger.Error().Err(err).Str("task_id", taskID).Msg("更新任务结果失败")
			_ = s.repo.UpdateTaskStatus(jobCtx, taskID, repository.TaskStatusFailed, err.Error())
			if s.metrics != nil {
				s.metrics.TaskTotal.WithLabelValues("failed").Inc()
			}
			return err
		}
		if err := s.repo.UpdateTaskStatus(jobCtx, taskID, repository.TaskStatusSuccess, ""); err != nil {
			s.logger.Error().Err(err).Str("task_id", taskID).Msg("更新任务状态失败")
			return err
		}
		if s.metrics != nil {
			s.metrics.TaskTotal.WithLabelValues("success").Inc()
		}
		s.logger.Info().
			Str("task_id", taskID).
			Str("output_path", outputPath).
			Msg("合并任务完成")
		return nil
	}

	if err := s.pool.Submit(job); err != nil {
		cleanup()
		if delErr := s.repo.DeleteTask(ctx, taskID); delErr != nil {
			s.logger.Error().Err(delErr).Str("task_id", taskID).Msg("删除任务记录失败")
		}
		s.logger.Warn().Err(err).Msg("提交任务到工作池失败")
		return nil, domainerrors.NewStorage("任务队列已满", "工作池无法接受任务", err)
	}

	result := buildResult(taskID, "")
	result.DownloadURL = fmt.Sprintf("%s/api/v1/files/%s/download", strings.TrimRight(baseURL, "/"), taskID)
	return result, nil
}

func normalizeMergeFormat(format string) (string, error) {
	if strings.TrimSpace(format) == "" {
		return mergeOutputFormat, nil
	}
	if strings.EqualFold(strings.TrimSpace(format), mergeOutputFormat) {
		return mergeOutputFormat, nil
	}
	return "", domainerrors.NewValidation("格式不支持", "合并仅支持 PDF 输出", nil)
}

func validateMergeSources(sources []Source) error {
	if len(sources) < 2 {
		return domainerrors.NewValidation("文件数量不足", "合并至少需要两个文件", nil)
	}
	return nil
}

func (s *ConverterService) prepareMergeSources(ctx context.Context, sources []Source) ([]Source, func(), error) {
	prepared := make([]Source, 0, len(sources))
	cleanups := make([]func(), 0, len(sources))
	for _, source := range sources {
		result, cleanup, err := s.prepareSource(ctx, source)
		if err != nil {
			for i := len(cleanups) - 1; i >= 0; i-- {
				cleanups[i]()
			}
			return nil, func() {}, err
		}
		prepared = append(prepared, result)
		cleanups = append(cleanups, cleanup)
	}
	return prepared, func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}, nil
}

func (s *ConverterService) mergeToPDF(ctx context.Context, sources []Source, outputDir string) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", domainerrors.NewStorage("创建输出目录失败", "无法创建合并输出目录", err)
	}

	workDir := filepath.Join(outputDir, "parts")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return "", domainerrors.NewStorage("创建临时目录失败", "无法创建合并临时目录", err)
	}
	defer os.RemoveAll(workDir)

	pdfPaths := make([]string, 0, len(sources))
	for idx, source := range sources {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if strings.EqualFold(filepath.Ext(source.FilePath), ".pdf") {
			pdfPaths = append(pdfPaths, source.FilePath)
			continue
		}

		partDir := filepath.Join(workDir, fmt.Sprintf("part-%02d", idx+1))
		if err := os.MkdirAll(partDir, 0o755); err != nil {
			return "", domainerrors.NewStorage("创建临时目录失败", "无法创建合并分片目录", err)
		}
		outputPath, err := s.executor.Convert(ctx, source.FilePath, partDir, mergeOutputFormat)
		if err != nil {
			return "", domainerrors.NewConversion("合并失败", "转换为 PDF 失败", err)
		}
		pdfPaths = append(pdfPaths, outputPath)
	}

	outputPath := filepath.Join(outputDir, "merged.pdf")
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := api.MergeCreateFile(pdfPaths, outputPath, false, nil); err != nil {
		return "", domainerrors.NewConversion("合并失败", "PDF 合并失败", err)
	}
	return outputPath, nil
}

func buildMergeSourceName(sources []Source) string {
	if len(sources) == 0 {
		return ""
	}
	first := sourceDisplayName(sources[0])
	summary := first
	if len(sources) > 1 {
		summary = fmt.Sprintf("%s 等 %d 个文件", first, len(sources))
	}
	return truncateByRunes(summary, 255)
}

func sourceDisplayName(source Source) string {
	if strings.TrimSpace(source.FileName) != "" {
		return source.FileName
	}
	if strings.TrimSpace(source.URL) != "" {
		return fileNameFromURL(source.URL)
	}
	return "未知文件"
}

func truncateByRunes(value string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}
