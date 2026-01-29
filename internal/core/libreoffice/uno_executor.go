package libreoffice

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// UnoExecutor UNO 执行器。
type UnoExecutor struct {
	Manager    *UnoManager
	PythonPath string
	ScriptPath string
	Timeout    time.Duration
	Logger     zerolog.Logger
}

// Convert 执行 UNO 转换。
func (e *UnoExecutor) Convert(ctx context.Context, inputPath, outputDir, format string) (string, error) {
	if e.Manager == nil {
		return "", fmt.Errorf("UNO 管理器未初始化")
	}
	ext, ok := ResolveFormat(format)
	if !ok || !IsUnoSupportedFormat(ext) {
		return "", fmt.Errorf("UNO 不支持的输出格式: %s", format)
	}
	if strings.TrimSpace(e.PythonPath) == "" {
		return "", fmt.Errorf("Python 路径不能为空")
	}
	if strings.TrimSpace(e.ScriptPath) == "" {
		return "", fmt.Errorf("UNO 脚本路径不能为空")
	}
	scriptPath := e.ScriptPath
	if !filepath.IsAbs(scriptPath) {
		abs, err := filepath.Abs(scriptPath)
		if err != nil {
			return "", fmt.Errorf("解析 UNO 脚本路径失败: %w", err)
		}
		scriptPath = abs
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("创建输出目录失败: %w", err)
	}

	base := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	outputPath := filepath.Join(outputDir, fmt.Sprintf("%s.%s", base, ext))

	inst, err := e.Manager.Acquire(ctx)
	if err != nil {
		return "", err
	}

	var convertErr error
	defer func() {
		e.Manager.Release(inst, convertErr)
	}()

	ctx, cancel := withTimeout(ctx, e.Timeout)
	defer cancel()

	args := []string{
		scriptPath,
		"--host", inst.host,
		"--port", fmt.Sprintf("%d", inst.port),
		"--input", inputPath,
		"--output", outputPath,
		"--format", ext,
	}

	cmd := exec.CommandContext(ctx, e.PythonPath, args...)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		convertErr = err
		return "", fmt.Errorf("UNO 转换失败: %w, 输出: %s", err, strings.TrimSpace(string(output)))
	}

	if _, err := os.Stat(outputPath); err != nil {
		convertErr = err
		return "", fmt.Errorf("UNO 输出文件不存在: %w", err)
	}

	return outputPath, nil
}

// CheckAvailable 检查 UNO 可用性。
func (e *UnoExecutor) CheckAvailable(ctx context.Context) error {
	if e.Manager == nil {
		return fmt.Errorf("UNO 管理器未初始化")
	}
	return e.Manager.CheckAvailable(ctx)
}
