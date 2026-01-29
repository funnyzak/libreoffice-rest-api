package http

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/funnyzak/libreoffice-rest-api/internal/infrastructure/config"
	"github.com/funnyzak/libreoffice-rest-api/internal/repository"
	"github.com/funnyzak/libreoffice-rest-api/internal/service"
	domainerrors "github.com/funnyzak/libreoffice-rest-api/pkg/errors"
)

// Handler HTTP 处理器。
type Handler struct {
	converter *service.ConverterService
	tasks     *service.TaskService
	repo      repository.Repository
	cfg       *config.Config
	logger    zerolog.Logger
}

// MergeRequest 合并请求体。
type MergeRequest struct {
	URLs   []string `json:"urls"`
	Format string   `json:"format"`
	Mode   string   `json:"mode"`
	Binary bool     `json:"binary"`
}

// NewHandler 创建处理器。
func NewHandler(converter *service.ConverterService, tasks *service.TaskService, repo repository.Repository, cfg *config.Config, logger zerolog.Logger) *Handler {
	return &Handler{
		converter: converter,
		tasks:     tasks,
		repo:      repo,
		cfg:       cfg,
		logger:    logger,
	}
}

// Convert 提交转换请求。
// @Summary 提交转换任务
// @Description 上传文件或 URL 提交转换任务，支持同步与异步模式。同步模式下可通过 binary 参数控制返回格式
// @Accept multipart/form-data
// @Accept json
// @Produce json
// @Param file formData file false "上传文件"
// @Param format formData string false "输出格式（常用：pdf/html/png/txt/odt/doc/docx/rtf/epub/xls/xlsx/ods/csv/ppt/pptx/odp/odg/svg/jpg/jpeg/webp）"
// @Param mode formData string false "sync/async，默认 async"
// @Param binary formData string false "同步模式下是否直接返回二进制，true/false，默认 true"
// @Param url formData string false "URL 下载地址"
// @Param body body map[string]string false "JSON 请求体"
// @Success 202 {object} APIResponse
// @Success 200 {object} APIResponse "同步模式且 binary=false 时返回 JSON"
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Router /api/v1/convert [post]
// @Security ApiKeyAuth
func (h *Handler) Convert(c *gin.Context) {
	maxBytes := minInt64(h.cfg.Storage.MaxFileMB*1024*1024, h.cfg.Server.MaxBodyMB*1024*1024)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)

	contentType := c.ContentType()
	mode := "async"
	format := ""
	h.logger.Debug().
		Str("content_type", contentType).
		Msg("开始处理转换请求")

	if strings.HasPrefix(contentType, "application/json") {
		var req struct {
			URL    string `json:"url"`
			Format string `json:"format"`
			Mode   string `json:"mode"`
			Binary bool   `json:"binary"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			h.writeError(c, domainerrors.NewValidation("请求参数错误", "无法解析 JSON", err), "解析 JSON 请求失败")
			return
		}
		mode = strings.ToLower(strings.TrimSpace(req.Mode))
		format = req.Format
		if req.URL == "" {
			h.writeError(c, domainerrors.NewValidation("URL 不能为空", "缺少 url 字段", nil), "URL 为空")
			return
		}
		source := service.Source{Type: service.SourceTypeURL, URL: req.URL}
		h.logger.Debug().
			Str("mode", mode).
			Str("format", format).
			Str("source_type", string(source.Type)).
			Str("url", source.URL).
			Bool("binary", req.Binary).
			Msg("解析转换请求成功")
		h.handleConvert(c, mode, format, source, req.Binary)
		return
	}

	mode = strings.ToLower(strings.TrimSpace(c.DefaultPostForm("mode", "async")))
	format = c.PostForm("format")
	binary := c.DefaultPostForm("binary", "true") == "true"

	if url := strings.TrimSpace(c.PostForm("url")); url != "" {
		source := service.Source{Type: service.SourceTypeURL, URL: url}
		h.logger.Debug().
			Str("mode", mode).
			Str("format", format).
			Str("source_type", string(source.Type)).
			Str("url", source.URL).
			Bool("binary", binary).
			Msg("解析转换请求成功")
		h.handleConvert(c, mode, format, source, binary)
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		h.writeError(c, domainerrors.NewValidation("文件不能为空", "未上传文件", err), "读取上传文件失败")
		return
	}
	defer file.Close()

	filePath, err := h.saveUploadToTemp(file, header.Filename)
	if err != nil {
		h.writeError(c, err, "保存上传文件失败")
		return
	}

	source := service.Source{
		Type:     service.SourceTypeUpload,
		FileName: header.Filename,
		FilePath: filePath,
		Size:     header.Size,
	}
	printable := strings.TrimSpace(format)
	if printable == "" {
		format = "pdf"
	}
	h.logger.Debug().
		Str("mode", mode).
		Str("format", format).
		Str("source_type", string(source.Type)).
		Str("file_name", source.FileName).
		Int64("size", source.Size).
		Bool("binary", binary).
		Msg("解析转换请求成功")
	h.handleConvert(c, mode, format, source, binary)
}

func (h *Handler) handleConvert(c *gin.Context, mode, format string, source service.Source, binary bool) {
	if mode == "" {
		mode = "async"
	}
	if strings.TrimSpace(format) == "" {
		format = "pdf"
	}
	baseURL := h.resolveBaseURL(c.Request)

	if mode == "sync" {
		result, err := h.converter.ConvertSync(c.Request.Context(), source, format)
		if err != nil {
			h.writeError(c, err, "同步转换失败")
			return
		}
		h.logger.Info().
			Str("mode", mode).
			Str("format", format).
			Str("output_ext", result.OutputExt).
			Str("output_name", result.OutputName).
			Bool("binary", binary).
			Msg("同步转换完成")

		if binary {
			defer os.RemoveAll(filepath.Dir(result.OutputPath))
			contentType := mime.TypeByExtension("." + result.OutputExt)
			if contentType == "" {
				contentType = "application/octet-stream"
			}
			c.Header("Content-Type", contentType)
			c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", result.OutputName))
			c.File(result.OutputPath)
			return
		}

		expiresAt := time.Now().Add(time.Duration(h.cfg.Storage.RetentionHours) * time.Hour)
		task := &repository.Task{
			ID:           uuid.New().String(),
			Status:       repository.TaskStatusSuccess,
			SourceType:   string(source.Type),
			SourceName:   source.FileName,
			SourceURL:    source.URL,
			InputPath:    source.FilePath,
			OutputPath:   result.OutputPath,
			OutputFormat: result.OutputExt,
			ExpiresAt:    expiresAt,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		if err := h.repo.CreateTask(c.Request.Context(), task); err != nil {
			os.RemoveAll(filepath.Dir(result.OutputPath))
			h.writeError(c, err, "创建任务失败")
			return
		}
		data := map[string]any{
			"task_id":       task.ID,
			"download_url":  buildDownloadURL(baseURL, task.ID),
			"output_name":   result.OutputName,
			"output_format": result.OutputExt,
		}
		WriteSuccess(c, http.StatusOK, data, "转换完成")
		return
	}

	result, err := h.converter.ConvertAsync(c.Request.Context(), source, format, baseURL)
	if err != nil {
		h.writeError(c, err, "提交异步转换失败")
		return
	}
	h.logger.Info().
		Str("mode", mode).
		Str("format", format).
		Str("task_id", result.TaskID).
		Msg("异步转换任务已提交")
	WriteSuccess(c, http.StatusAccepted, result, "任务已提交")
}

// Merge 提交合并请求。
// @Summary 提交合并任务
// @Description 上传多个文件或 URL 合并为单个 PDF，支持同步与异步模式。同步模式下可通过 binary 参数控制返回格式
// @Accept multipart/form-data
// @Accept json
// @Produce json
// @Param files formData file false "上传文件（可多次传入）"
// @Param format formData string false "输出格式（仅支持 pdf）"
// @Param mode formData string false "sync/async，默认 async"
// @Param binary formData string false "同步模式下是否直接返回二进制，true/false，默认 true"
// @Param urls formData []string false "URL 列表"
// @Param body body MergeRequest false "JSON 请求体"
// @Success 202 {object} APIResponse
// @Success 200 {object} APIResponse "同步模式且 binary=false 时返回 JSON"
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Router /api/v1/merge [post]
// @Security ApiKeyAuth
func (h *Handler) Merge(c *gin.Context) {
	maxBytes := h.cfg.Server.MaxBodyMB * 1024 * 1024
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)

	contentType := c.ContentType()
	mode := "async"
	format := ""
	h.logger.Debug().
		Str("content_type", contentType).
		Msg("开始处理合并请求")

	if strings.HasPrefix(contentType, "application/json") {
		var req MergeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			h.writeError(c, domainerrors.NewValidation("请求参数错误", "无法解析 JSON", err), "解析 JSON 请求失败")
			return
		}
		mode = strings.ToLower(strings.TrimSpace(req.Mode))
		format = req.Format
		binary := req.Binary
		if len(req.URLs) == 0 {
			h.writeError(c, domainerrors.NewValidation("URL 不能为空", "缺少 urls 字段", nil), "URL 为空")
			return
		}
		sources := make([]service.Source, 0, len(req.URLs))
		for _, rawURL := range req.URLs {
			trimmed := strings.TrimSpace(rawURL)
			if trimmed == "" {
				h.writeError(c, domainerrors.NewValidation("URL 不能为空", "URL 列表包含空值", nil), "URL 为空")
				return
			}
			sources = append(sources, service.Source{Type: service.SourceTypeURL, URL: trimmed})
		}
		if len(sources) < 2 {
			h.writeError(c, domainerrors.NewValidation("文件数量不足", "合并至少需要两个文件", nil), "合并文件数量不足")
			return
		}
		h.handleMerge(c, mode, format, sources, binary)
		return
	}

	mode = strings.ToLower(strings.TrimSpace(c.DefaultPostForm("mode", "async")))
	format = c.PostForm("format")
	binary := c.DefaultPostForm("binary", "true") == "true"

	urls := expandFormURLs(c.PostFormArray("urls"))
	if len(urls) == 0 {
		urls = expandFormURLs(c.PostFormArray("url"))
	}
	if len(urls) > 0 {
		form, err := c.MultipartForm()
		if err == nil && form != nil && len(form.File) > 0 {
			h.writeError(c, domainerrors.NewValidation("参数冲突", "不能同时上传文件与 URL", nil), "合并请求参数冲突")
			return
		}
		if len(urls) < 2 {
			h.writeError(c, domainerrors.NewValidation("文件数量不足", "合并至少需要两个文件", nil), "合并文件数量不足")
			return
		}
		sources := make([]service.Source, 0, len(urls))
		for _, rawURL := range urls {
			trimmed := strings.TrimSpace(rawURL)
			if trimmed == "" {
				h.writeError(c, domainerrors.NewValidation("URL 不能为空", "URL 列表包含空值", nil), "URL 为空")
				return
			}
			sources = append(sources, service.Source{Type: service.SourceTypeURL, URL: trimmed})
		}
		h.handleMerge(c, mode, format, sources, binary)
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		h.writeError(c, domainerrors.NewValidation("文件不能为空", "未上传文件", err), "读取上传文件失败")
		return
	}
	files := form.File["files"]
	if len(files) == 0 {
		files = form.File["file"]
	}
	if len(files) == 0 {
		h.writeError(c, domainerrors.NewValidation("文件不能为空", "未上传文件", nil), "读取上传文件失败")
		return
	}
	if len(files) < 2 {
		h.writeError(c, domainerrors.NewValidation("文件数量不足", "合并至少需要两个文件", nil), "合并文件数量不足")
		return
	}
	sources := make([]service.Source, 0, len(files))
	cleanup := func() {
		for _, source := range sources {
			if source.FilePath == "" {
				continue
			}
			_ = os.Remove(source.FilePath)
		}
	}
	for _, header := range files {
		file, err := header.Open()
		if err != nil {
			cleanup()
			h.writeError(c, domainerrors.NewStorage("读取文件失败", "无法打开上传文件", err), "读取上传文件失败")
			return
		}
		filePath, err := h.saveUploadToTemp(file, header.Filename)
		_ = file.Close()
		if err != nil {
			cleanup()
			h.writeError(c, err, "保存上传文件失败")
			return
		}
		sources = append(sources, service.Source{
			Type:     service.SourceTypeUpload,
			FileName: header.Filename,
			FilePath: filePath,
			Size:     header.Size,
		})
	}
	h.handleMerge(c, mode, format, sources, binary)
}

func (h *Handler) handleMerge(c *gin.Context, mode, format string, sources []service.Source, binary bool) {
	if mode == "" {
		mode = "async"
	}
	if strings.TrimSpace(format) == "" {
		format = "pdf"
	}
	baseURL := h.resolveBaseURL(c.Request)

	if mode == "sync" {
		result, err := h.converter.MergeSync(c.Request.Context(), sources, format)
		if err != nil {
			h.writeError(c, err, "同步合并失败")
			return
		}
		h.logger.Info().
			Str("mode", mode).
			Str("output_ext", result.OutputExt).
			Str("output_name", result.OutputName).
			Bool("binary", binary).
			Msg("同步合并完成")

		if binary {
			defer os.RemoveAll(filepath.Dir(result.OutputPath))
			contentType := mime.TypeByExtension("." + result.OutputExt)
			if contentType == "" {
				contentType = "application/octet-stream"
			}
			c.Header("Content-Type", contentType)
			c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", result.OutputName))
			c.File(result.OutputPath)
			return
		}

		expiresAt := time.Now().Add(time.Duration(h.cfg.Storage.RetentionHours) * time.Hour)
		task := &repository.Task{
			ID:           uuid.New().String(),
			Status:       repository.TaskStatusSuccess,
			SourceType:   string(service.SourceTypeUpload),
			SourceName:   "merge",
			InputPath:    "",
			OutputPath:   result.OutputPath,
			OutputFormat: result.OutputExt,
			ExpiresAt:    expiresAt,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		if err := h.repo.CreateTask(c.Request.Context(), task); err != nil {
			os.RemoveAll(filepath.Dir(result.OutputPath))
			h.writeError(c, err, "创建任务失败")
			return
		}
		data := map[string]any{
			"task_id":       task.ID,
			"download_url":  buildDownloadURL(baseURL, task.ID),
			"output_name":   result.OutputName,
			"output_format": result.OutputExt,
		}
		WriteSuccess(c, http.StatusOK, data, "合并完成")
		return
	}

	result, err := h.converter.MergeAsync(c.Request.Context(), sources, format, baseURL)
	if err != nil {
		h.writeError(c, err, "提交异步合并失败")
		return
	}
	h.logger.Info().
		Str("mode", mode).
		Str("task_id", result.TaskID).
		Int("source_count", len(sources)).
		Msg("异步合并任务已提交")
	WriteSuccess(c, http.StatusAccepted, result, "任务已提交")
}

// GetTask 获取任务状态。
// @Summary 查询任务状态
// @Produce json
// @Param id path string true "任务 ID"
// @Success 200 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Router /api/v1/tasks/{id} [get]
// @Security ApiKeyAuth
func (h *Handler) GetTask(c *gin.Context) {
	id := c.Param("id")
	h.logger.Debug().Str("task_id", id).Msg("查询任务状态")
	task, err := h.tasks.GetTask(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err, "查询任务失败")
		return
	}
	data := map[string]any{
		"id":            task.ID,
		"status":        task.Status,
		"source_type":   task.SourceType,
		"source_name":   task.SourceName,
		"output_format": task.OutputFormat,
		"error":         task.ErrorMessage,
		"created_at":    task.CreatedAt,
		"updated_at":    task.UpdatedAt,
		"expires_at":    task.ExpiresAt,
	}
	if task.Status == repository.TaskStatusSuccess {
		data["download_url"] = buildDownloadURL(h.resolveBaseURL(c.Request), task.ID)
	}
	h.logger.Info().
		Str("task_id", task.ID).
		Str("status", string(task.Status)).
		Msg("任务状态查询成功")
	WriteSuccess(c, http.StatusOK, data, "查询成功")
}

// Download 下载文件。
// @Summary 下载转换结果
// @Param id path string true "任务 ID"
// @Success 200 {string} file
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Router /api/v1/files/{id}/download [get]
// @Security ApiKeyAuth
func (h *Handler) Download(c *gin.Context) {
	id := c.Param("id")
	h.logger.Debug().Str("task_id", id).Msg("下载文件请求")
	task, err := h.tasks.GetTask(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err, "获取任务失败")
		return
	}
	if task.Status != repository.TaskStatusSuccess {
		h.writeError(c, domainerrors.NewValidation("文件未就绪", "任务尚未完成", nil), "任务未完成")
		return
	}
	if task.OutputPath == "" {
		h.writeError(c, domainerrors.NewNotFound("文件不存在", "未找到输出文件", nil), "输出文件为空")
		return
	}
	if _, err := os.Stat(task.OutputPath); err != nil {
		h.writeError(c, domainerrors.NewNotFound("文件不存在", "输出文件丢失", err), "输出文件丢失")
		return
	}
	h.logger.Info().
		Str("task_id", task.ID).
		Str("file_name", filepath.Base(task.OutputPath)).
		Msg("开始下载文件")
	c.FileAttachment(task.OutputPath, filepath.Base(task.OutputPath))
}

// Health 健康检查。
// @Summary 健康检查
// @Produce json
// @Success 200 {object} APIResponse
// @Router /health [get]
// @Security ApiKeyAuth
func (h *Handler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	status, checks := h.checkHealth(ctx)
	data := map[string]any{
		"status": status,
		"checks": checks,
		"time":   time.Now(),
	}
	if status == "healthy" {
		h.logger.Debug().Msg("健康检查通过")
	} else {
		h.logger.Warn().
			Str("status", status).
			Msg("健康检查异常")
	}
	WriteSuccess(c, http.StatusOK, data, "健康检查")
}

func (h *Handler) checkHealth(ctx context.Context) (string, map[string]any) {
	checks := make(map[string]any)
	status := "healthy"

	if err := h.repo.Ping(ctx); err != nil {
		checks["database"] = "unhealthy"
		status = "unhealthy"
	} else {
		checks["database"] = "healthy"
	}

	freeGB, err := diskFreeGB(h.cfg.Storage.OutputDir)
	if err != nil {
		checks["disk"] = "unhealthy"
		status = "unhealthy"
	} else if freeGB < h.cfg.Health.MinFreeGB {
		checks["disk"] = fmt.Sprintf("degraded: free %dGB", freeGB)
		if status == "healthy" {
			status = "degraded"
		}
	} else {
		checks["disk"] = fmt.Sprintf("healthy: free %dGB", freeGB)
	}

	if err := h.converter.CheckAvailable(ctx); err != nil {
		checks["libreoffice"] = "unhealthy"
		status = "unhealthy"
	} else {
		checks["libreoffice"] = "healthy"
	}

	return status, checks
}

func (h *Handler) writeError(c *gin.Context, err error, msg string) {
	h.logRequestError(c, err, msg)
	WriteError(c, err)
}

func (h *Handler) logRequestError(c *gin.Context, err error, msg string) {
	event := h.logger.Error()
	var de *domainerrors.DomainError
	if ok := domainerrors.AsDomainError(err, &de); ok {
		switch de.Type {
		case domainerrors.ErrorValidation, domainerrors.ErrorAuthentication, domainerrors.ErrorNotFound:
			event = h.logger.Warn()
		}
	}
	event.
		Err(err).
		Str("method", c.Request.Method).
		Str("path", c.Request.URL.Path).
		Msg(msg)
}

func getBaseURL(req *http.Request) string {
	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, req.Host)
}

func (h *Handler) resolveBaseURL(req *http.Request) string {
	if h == nil || h.cfg == nil {
		return getBaseURL(req)
	}
	publicBaseURL := strings.TrimSpace(h.cfg.Server.PublicBaseURL)
	if publicBaseURL == "" {
		return getBaseURL(req)
	}
	return publicBaseURL
}

func buildDownloadURL(baseURL, taskID string) string {
	return fmt.Sprintf("%s/api/v1/files/%s/download", strings.TrimRight(baseURL, "/"), taskID)
}

func ioCopy(dst io.Writer, src io.Reader) (int64, error) {
	return io.Copy(dst, src)
}

func expandFormURLs(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	expanded := make([]string, 0, len(values))
	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			trimmed := strings.TrimSpace(item)
			if trimmed == "" {
				continue
			}
			expanded = append(expanded, trimmed)
		}
	}
	return expanded
}

func (h *Handler) saveUploadToTemp(reader io.Reader, filename string) (string, error) {
	fileID := uuid.New().String()
	filePath := filepath.Join(h.cfg.Storage.TempDir, fileID+filepath.Ext(filename))
	out, err := os.Create(filePath)
	if err != nil {
		return "", domainerrors.NewStorage("保存文件失败", "无法创建临时文件", err)
	}
	if _, err := ioCopy(out, reader); err != nil {
		_ = out.Close()
		_ = os.Remove(filePath)
		return "", domainerrors.NewStorage("保存文件失败", "写入临时文件失败", err)
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(filePath)
		return "", domainerrors.NewStorage("保存文件失败", "关闭临时文件失败", err)
	}
	return filePath, nil
}

func minInt64(a, b int64) int64 {
	if a <= 0 {
		return b
	}
	if b <= 0 {
		return a
	}
	if a < b {
		return a
	}
	return b
}
