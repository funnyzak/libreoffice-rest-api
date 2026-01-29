package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/funnyzak/libreoffice-rest-api/internal/core/libreoffice"
	"github.com/funnyzak/libreoffice-rest-api/internal/core/workerpool"
	"github.com/funnyzak/libreoffice-rest-api/internal/infrastructure/config"
	metrics "github.com/funnyzak/libreoffice-rest-api/internal/infrastructure/metrics"
	"github.com/funnyzak/libreoffice-rest-api/internal/repository"
	domainerrors "github.com/funnyzak/libreoffice-rest-api/pkg/errors"
)

// SourceType 来源类型。
type SourceType string

const (
	// SourceTypeUpload 上传文件。
	SourceTypeUpload SourceType = "upload"
	// SourceTypeURL URL 下载。
	SourceTypeURL SourceType = "url"
)

// Source 转换源。
type Source struct {
	Type     SourceType
	FileName string
	FilePath string
	URL      string
	Size     int64
}

// ConvertResult 转换结果。
type ConvertResult struct {
	TaskID      string `json:"task_id,omitempty"`
	OutputPath  string `json:"output_path,omitempty"`
	OutputName  string `json:"output_name,omitempty"`
	OutputExt   string `json:"output_ext,omitempty"`
	DownloadURL string `json:"download_url,omitempty"`
}

// ConverterService 转换服务。
type ConverterService struct {
	repo     repository.Repository
	executor libreoffice.Executor
	pool     *workerpool.Pool
	cfg      *config.Config
	metrics  *metrics.Metrics
	logger   zerolog.Logger
}

// NewConverterService 创建转换服务。
func NewConverterService(repo repository.Repository, executor libreoffice.Executor, pool *workerpool.Pool, cfg *config.Config, metricsCollector *metrics.Metrics, logger zerolog.Logger) *ConverterService {
	return &ConverterService{
		repo:     repo,
		executor: executor,
		pool:     pool,
		cfg:      cfg,
		metrics:  metricsCollector,
		logger:   logger,
	}
}

// CheckAvailable 检查 LibreOffice 可用性。
func (s *ConverterService) CheckAvailable(ctx context.Context) error {
	return s.executor.CheckAvailable(ctx)
}

// ConvertSync 同步转换。
func (s *ConverterService) ConvertSync(ctx context.Context, source Source, format string) (*ConvertResult, error) {
	s.logger.Debug().
		Str("source_type", string(source.Type)).
		Str("format", format).
		Str("file_name", source.FileName).
		Msg("开始同步转换")
	prepared, cleanup, err := s.prepareSource(ctx, source)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	if err := s.validateFormat(format); err != nil {
		return nil, err
	}

	outputDir := filepath.Join(s.cfg.Storage.OutputDir, uuid.New().String())
	outputPath, err := s.executor.Convert(ctx, prepared.FilePath, outputDir, format)
	if err != nil {
		return nil, domainerrors.NewConversion("转换失败", "LibreOffice 执行失败", err)
	}

	result := buildResult("", outputPath)
	s.logger.Info().
		Str("output_ext", result.OutputExt).
		Str("output_name", result.OutputName).
		Msg("同步转换完成")
	return result, nil
}

// ConvertAsync 异步转换。
func (s *ConverterService) ConvertAsync(ctx context.Context, source Source, format string, baseURL string) (*ConvertResult, error) {
	s.logger.Debug().
		Str("source_type", string(source.Type)).
		Str("format", format).
		Str("file_name", source.FileName).
		Msg("开始异步转换")
	prepared, cleanup, err := s.prepareSource(ctx, source)
	if err != nil {
		return nil, err
	}

	if err := s.validateFormat(format); err != nil {
		cleanup()
		return nil, err
	}

	taskID := uuid.New().String()
	expiresAt := time.Now().Add(time.Duration(s.cfg.Storage.RetentionHours) * time.Hour)

	task := &repository.Task{
		ID:           taskID,
		Status:       repository.TaskStatusPending,
		SourceType:   string(prepared.Type),
		SourceName:   prepared.FileName,
		SourceURL:    prepared.URL,
		InputPath:    prepared.FilePath,
		OutputFormat: strings.ToLower(format),
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
		Str("source_type", task.SourceType).
		Str("output_format", task.OutputFormat).
		Time("expires_at", task.ExpiresAt).
		Msg("异步任务已创建")

	job := func(jobCtx context.Context) error {
		defer cleanup()
		if s.metrics != nil {
			s.metrics.ActiveTasks.Inc()
			defer s.metrics.ActiveTasks.Dec()
		}
		s.logger.Debug().
			Str("task_id", taskID).
			Msg("开始执行转换任务")
		if err := s.repo.UpdateTaskStatus(jobCtx, taskID, repository.TaskStatusProcessing, ""); err != nil {
			s.logger.Error().Err(err).Msg("更新任务状态失败")
		}

		outputDir := filepath.Join(s.cfg.Storage.OutputDir, taskID)
		outputPath, err := s.executor.Convert(jobCtx, prepared.FilePath, outputDir, format)
		if err != nil {
			s.logger.Error().Err(err).Str("task_id", taskID).Msg("转换执行失败")
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
			Msg("转换任务完成")
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

func (s *ConverterService) prepareSource(ctx context.Context, source Source) (Source, func(), error) {
	switch source.Type {
	case SourceTypeUpload:
		if source.FilePath == "" {
			return Source{}, func() {}, domainerrors.NewValidation("文件路径不能为空", "上传文件未保存", nil)
		}
		if err := s.validateFile(source.FilePath, source.FileName); err != nil {
			return Source{}, func() {}, err
		}
		cleanup := func() {
			_ = os.Remove(source.FilePath)
		}
		return source, cleanup, nil
	case SourceTypeURL:
		return s.downloadSource(ctx, source)
	default:
		return Source{}, func() {}, domainerrors.NewValidation("来源类型不合法", "未识别的来源类型", nil)
	}
}

func (s *ConverterService) downloadSource(ctx context.Context, source Source) (Source, func(), error) {
	if err := s.validateURL(ctx, source.URL); err != nil {
		return Source{}, func() {}, err
	}
	s.logger.Debug().
		Str("url", source.URL).
		Msg("开始下载源文件")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.URL, nil)
	if err != nil {
		return Source{}, func() {}, domainerrors.NewValidation("URL 不合法", "无法创建下载请求", err)
	}
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return domainerrors.NewValidation("URL 不合法", "重定向次数过多", nil)
			}
			if err := s.validateURL(ctx, req.URL.String()); err != nil {
				return err
			}
			return nil
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return Source{}, func() {}, domainerrors.NewConversion("下载失败", "无法下载源文件", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.logger.Warn().
			Str("url", source.URL).
			Int("status", resp.StatusCode).
			Msg("下载源文件失败")
		return Source{}, func() {}, domainerrors.NewConversion("下载失败", fmt.Sprintf("状态码: %d", resp.StatusCode), nil)
	}

	fileName := source.FileName
	if fileName == "" {
		fileName = fileNameFromDisposition(resp.Header.Get("Content-Disposition"))
		if fileName == "" {
			fileName = fileNameFromURL(source.URL)
		}
	}

	if len([]rune(fileName)) > s.cfg.Security.MaxFilenameLength {
		s.logger.Warn().
			Str("file_name", fileName).
			Int("max_length", s.cfg.Security.MaxFilenameLength).
			Msg("下载文件名超出限制")
		return Source{}, func() {}, domainerrors.NewValidation("文件名过长", "下载文件名超出限制", nil)
	}

	fileID := uuid.New().String()
	filePath := filepath.Join(s.cfg.Storage.TempDir, fileID+filepath.Ext(fileName))
	file, err := os.Create(filePath)
	if err != nil {
		return Source{}, func() {}, domainerrors.NewStorage("保存文件失败", "无法创建临时文件", err)
	}
	defer file.Close()

	limit := s.cfg.Storage.MaxFileMB * 1024 * 1024
	reader := io.LimitReader(resp.Body, limit+1)
	written, err := io.Copy(file, reader)
	if err != nil {
		_ = os.Remove(filePath)
		return Source{}, func() {}, domainerrors.NewStorage("保存文件失败", "写入临时文件失败", err)
	}
	if written > limit {
		s.logger.Warn().
			Int64("size", written).
			Int64("max_size", limit).
			Msg("下载文件超过大小限制")
		_ = os.Remove(filePath)
		return Source{}, func() {}, domainerrors.NewValidation("文件过大", "超过允许大小", nil)
	}

	downloaded := Source{
		Type:     SourceTypeURL,
		FileName: fileName,
		FilePath: filePath,
		URL:      source.URL,
		Size:     written,
	}
	if err := s.validateFile(filePath, fileName); err != nil {
		_ = os.Remove(filePath)
		return Source{}, func() {}, err
	}

	cleanup := func() {
		_ = os.Remove(filePath)
	}
	s.logger.Info().
		Str("file_name", fileName).
		Int64("size", written).
		Msg("下载源文件完成")
	return downloaded, cleanup, nil
}

func (s *ConverterService) validateFormat(format string) error {
	if _, ok := libreoffice.ResolveFormat(format); ok {
		return nil
	}
	return domainerrors.NewValidation("格式不支持", "请使用支持的输出格式", nil)
}

func (s *ConverterService) validateFile(path, name string) error {
	info, err := os.Stat(path)
	if err != nil {
		return domainerrors.NewValidation("文件不存在", "无法读取文件信息", err)
	}
	if info.Size() == 0 {
		return domainerrors.NewValidation("文件为空", "文件大小为 0", nil)
	}
	if info.Size() > s.cfg.Storage.MaxFileMB*1024*1024 {
		return domainerrors.NewValidation("文件过大", "超过允许大小", nil)
	}
	if len([]rune(name)) > s.cfg.Security.MaxFilenameLength {
		return domainerrors.NewValidation("文件名过长", "超出长度限制", nil)
	}
	if !s.allowedExtension(name) {
		return domainerrors.NewValidation("文件扩展名不允许", "扩展名不在白名单", nil)
	}

	file, err := os.Open(path)
	if err != nil {
		return domainerrors.NewValidation("文件不可读", "无法打开文件", err)
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return domainerrors.NewValidation("文件不可读", "无法读取文件头", err)
	}
	mimeType := http.DetectContentType(buf[:n])
	if !s.allowedMime(mimeType) {
		if !s.allowZipMime(name, mimeType) {
			return domainerrors.NewValidation("文件类型不允许", fmt.Sprintf("检测到类型: %s", mimeType), nil)
		}
	}

	return nil
}

func (s *ConverterService) allowedMime(mimeType string) bool {
	for _, allowed := range s.cfg.Security.AllowedMIMETypes {
		if strings.EqualFold(allowed, mimeType) {
			return true
		}
	}
	return false
}

func (s *ConverterService) validateURL(ctx context.Context, rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return domainerrors.NewValidation("URL 不合法", "无法解析 URL", err)
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return domainerrors.NewValidation("URL 不合法", "仅支持 http/https", nil)
	}

	if parsed.User != nil {
		return domainerrors.NewValidation("URL 不合法", "不允许包含用户信息", nil)
	}

	host := parsed.Host
	if strings.Contains(host, ":") {
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
	}
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return domainerrors.NewValidation("URL 不合法", "Host 为空", nil)
	}
	if strings.EqualFold(host, "localhost") || strings.HasSuffix(strings.ToLower(host), ".localhost") {
		return domainerrors.NewValidation("URL 不合法", "禁止访问本地地址", nil)
	}

	if ip := net.ParseIP(host); ip != nil {
		if !isPublicIP(ip) {
			return domainerrors.NewValidation("URL 不合法", "禁止访问内网地址", nil)
		}
		return nil
	}

	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil || len(ips) == 0 {
		return domainerrors.NewValidation("URL 不合法", "域名解析失败", err)
	}
	for _, addr := range ips {
		if !isPublicIP(addr.IP) {
			return domainerrors.NewValidation("URL 不合法", "禁止访问内网地址", nil)
		}
	}
	return nil
}

func isPublicIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return false
	}
	return true
}

func (s *ConverterService) allowedExtension(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	for _, allowed := range s.cfg.Security.AllowedExtensions {
		if strings.EqualFold(allowed, ext) {
			return true
		}
	}
	return false
}

func (s *ConverterService) allowZipMime(name, mimeType string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	if strings.EqualFold(mimeType, "application/zip") {
		switch ext {
		case ".docx", ".pptx", ".xlsx", ".odt", ".ods", ".odp":
			return true
		default:
			return false
		}
	}
	if strings.EqualFold(mimeType, "application/octet-stream") {
		switch ext {
		case ".doc", ".xls", ".ppt", ".docx", ".pptx", ".xlsx", ".odt", ".ods", ".odp":
			return true
		default:
			return false
		}
	}
	return false
}

func fileNameFromDisposition(contentDisposition string) string {
	if contentDisposition == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(contentDisposition)
	if err != nil {
		return ""
	}
	return params["filename"]
}

func fileNameFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Path == "" {
		return "download"
	}
	base := path.Base(parsed.Path)
	if base == "." || base == "/" || base == "" {
		return "download"
	}
	return base
}

func buildResult(taskID, outputPath string) *ConvertResult {
	result := &ConvertResult{
		TaskID: taskID,
	}
	if outputPath == "" {
		return result
	}
	result.OutputPath = outputPath
	result.OutputExt = strings.TrimPrefix(filepath.Ext(outputPath), ".")
	result.OutputName = filepath.Base(outputPath)
	return result
}

// HashFile 计算文件哈希。
func HashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
