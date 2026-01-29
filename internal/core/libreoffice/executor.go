package libreoffice

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Executor LibreOffice 执行器接口。
type Executor interface {
	Convert(ctx context.Context, inputPath, outputDir, format string) (string, error)
	CheckAvailable(ctx context.Context) error
}

// CommandExecutor 命令行执行器。
type CommandExecutor struct {
	LibreOfficePath    string
	UserProfileBaseDir string
	Timeout            time.Duration
}

var allowedFormats = map[string]string{
	"csv":   "csv",
	"doc":   "doc",
	"docx":  "docx",
	"epub":  "epub",
	"htm":   "htm",
	"html":  "html",
	"jpeg":  "jpeg",
	"jpg":   "jpg",
	"odg":   "odg",
	"odp":   "odp",
	"ods":   "ods",
	"odt":   "odt",
	"pdf":   "pdf",
	"png":   "png",
	"ppt":   "ppt",
	"pptx":  "pptx",
	"rtf":   "rtf",
	"svg":   "svg",
	"tab":   "tab",
	"tsv":   "tsv",
	"txt":   "txt",
	"webp":  "webp",
	"xhtml": "xhtml",
	"xls":   "xls",
	"xlsx":  "xlsx",
}

// Convert 执行 LibreOffice 转换。
func (e *CommandExecutor) Convert(ctx context.Context, inputPath, outputDir, format string) (string, error) {
	ext, ok := ResolveFormat(format)
	if !ok {
		return "", fmt.Errorf("不支持的输出格式: %s", format)
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("创建输出目录失败: %w", err)
	}

	userProfileDir := filepath.Join(e.UserProfileBaseDir, uuid.New().String())
	if err := os.MkdirAll(userProfileDir, 0o755); err != nil {
		return "", fmt.Errorf("创建用户目录失败: %w", err)
	}
	defer os.RemoveAll(userProfileDir)

	userProfileURL, err := fileURLFromPath(userProfileDir)
	if err != nil {
		return "", fmt.Errorf("构建用户目录 URL 失败: %w", err)
	}

	ctx, cancel := withTimeout(ctx, e.Timeout)
	defer cancel()

	args := []string{
		"--headless",
		"--invisible",
		"--nologo",
		"--nolockcheck",
		"--nodefault",
		"--nofirststartwizard",
		"--nocrashreport",
		"--norestore",
		fmt.Sprintf("-env:UserInstallation=%s", userProfileURL),
		"--convert-to",
		ext,
		"--outdir",
		outputDir,
		inputPath,
	}

	cmd := exec.CommandContext(ctx, e.LibreOfficePath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("执行转换失败: %w, 输出: %s", err, string(output))
	}

	outputFile, err := findOutputFile(outputDir, inputPath, ext)
	if err != nil {
		return "", err
	}

	return outputFile, nil
}

// ResolveFormat 解析并校验输出格式，返回对应扩展名。
func ResolveFormat(format string) (string, bool) {
	format = strings.ToLower(strings.TrimSpace(format))
	ext, ok := allowedFormats[format]
	return ext, ok
}

// SupportedFormats 返回支持的输出格式列表。
func SupportedFormats() []string {
	formats := make([]string, 0, len(allowedFormats))
	for key := range allowedFormats {
		formats = append(formats, key)
	}
	sort.Strings(formats)
	return formats
}

// CheckAvailable 检查 LibreOffice 是否可用。
func (e *CommandExecutor) CheckAvailable(ctx context.Context) error {
	ctx, cancel := withTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, e.LibreOfficePath, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("LibreOffice 不可用: %w", err)
	}
	return nil
}

func withTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return context.WithCancel(ctx)
	}
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining <= timeout {
			return context.WithCancel(ctx)
		}
	}
	return context.WithTimeout(ctx, timeout)
}

func findOutputFile(outputDir, inputPath, ext string) (string, error) {
	base := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	candidate := filepath.Join(outputDir, fmt.Sprintf("%s.%s", base, ext))
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}
	matches, err := filepath.Glob(filepath.Join(outputDir, "*"))
	if err != nil {
		return "", fmt.Errorf("查找输出文件失败: %w", err)
	}
	baseLower := strings.ToLower(base)
	for _, match := range matches {
		name := strings.TrimSuffix(filepath.Base(match), filepath.Ext(match))
		nameLower := strings.ToLower(name)
		extMatch := strings.EqualFold(strings.TrimPrefix(filepath.Ext(match), "."), ext)
		baseMatch := strings.EqualFold(name, base) || strings.HasPrefix(nameLower, baseLower+"-")
		if extMatch && baseMatch {
			return match, nil
		}
	}
	return "", errors.New("未找到转换输出文件")
}

func fileURLFromPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	filePath := filepath.ToSlash(abs)
	if volume := filepath.VolumeName(abs); volume != "" && !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}
	u := url.URL{
		Scheme: "file",
		Path:   filePath,
	}
	return u.String(), nil
}
