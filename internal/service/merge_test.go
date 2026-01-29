package service

import (
	"strings"
	"testing"
)

func TestNormalizeMergeFormat(t *testing.T) {
	t.Parallel()

	format, err := normalizeMergeFormat("")
	if err != nil {
		t.Fatalf("期望默认格式成功: %v", err)
	}
	if format != "pdf" {
		t.Fatalf("期望默认格式为 pdf")
	}

	format, err = normalizeMergeFormat("PDF")
	if err != nil {
		t.Fatalf("期望支持 PDF: %v", err)
	}
	if format != "pdf" {
		t.Fatalf("期望返回标准格式 pdf")
	}

	if _, err := normalizeMergeFormat("docx"); err == nil {
		t.Fatalf("期望不支持格式返回错误")
	}
}

func TestBuildMergeSourceName(t *testing.T) {
	t.Parallel()

	name := buildMergeSourceName([]Source{
		{FileName: "a.docx"},
		{FileName: "b.docx"},
	})
	if name != "a.docx 等 2 个文件" {
		t.Fatalf("合并摘要不符合预期: %s", name)
	}

	name = buildMergeSourceName([]Source{
		{URL: "https://example.com/demo.docx"},
		{FileName: "b.docx"},
	})
	if name != "demo.docx 等 2 个文件" {
		t.Fatalf("URL 文件名解析失败: %s", name)
	}
}

func TestTruncateByRunes(t *testing.T) {
	t.Parallel()

	value := strings.Repeat("a", 300)
	result := truncateByRunes(value, 255)
	if len(result) > 255 {
		t.Fatalf("期望截断后长度不超过 255")
	}
	if !strings.HasSuffix(result, "...") {
		t.Fatalf("期望使用省略号")
	}
}
