package libreoffice

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFindOutputFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	input := filepath.Join(tmpDir, "demo.docx")
	_ = os.WriteFile(input, []byte("data"), 0o644)

	output := filepath.Join(tmpDir, "demo.pdf")
	_ = os.WriteFile(output, []byte("out"), 0o644)

	found, err := findOutputFile(tmpDir, input, "pdf")
	if err != nil {
		t.Fatalf("查找输出文件失败: %v", err)
	}
	if found != output {
		t.Fatalf("输出路径不匹配")
	}
}

func TestFindOutputFileMultiPagePNG(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	input := filepath.Join(tmpDir, "demo.pptx")
	_ = os.WriteFile(input, []byte("data"), 0o644)

	output := filepath.Join(tmpDir, "demo-1.png")
	_ = os.WriteFile(output, []byte("out"), 0o644)

	found, err := findOutputFile(tmpDir, input, "png")
	if err != nil {
		t.Fatalf("查找输出文件失败: %v", err)
	}
	if found != output {
		t.Fatalf("输出路径不匹配")
	}
}

func TestConvertUnsupportedFormat(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	input := filepath.Join(tmpDir, "demo.docx")
	_ = os.WriteFile(input, []byte("data"), 0o644)

	executor := &CommandExecutor{LibreOfficePath: "soffice", UserProfileBaseDir: tmpDir, Timeout: time.Second}
	if _, err := executor.Convert(context.Background(), input, tmpDir, "txt"); err == nil {
		t.Fatalf("期望不支持格式错误")
	}
}

func TestCheckAvailableInvalidPath(t *testing.T) {
	t.Parallel()

	executor := &CommandExecutor{LibreOfficePath: "not-exist", UserProfileBaseDir: t.TempDir(), Timeout: time.Second}
	if err := executor.CheckAvailable(context.Background()); err == nil {
		t.Fatalf("期望检测失败")
	}
}
