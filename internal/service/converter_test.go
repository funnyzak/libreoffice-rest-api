package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"

	"github.com/funnyzak/libreoffice-rest-api/internal/infrastructure/config"
)

func TestValidateFileSmallFile(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Storage: config.StorageConfig{MaxFileMB: 1},
		Security: config.SecurityConfig{
			AllowedExtensions: []string{".docx"},
			MaxFilenameLength: 128,
		},
	}
	service := &ConverterService{cfg: cfg, logger: zerolog.Nop()}

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "sample.docx")
	if err := os.WriteFile(filePath, []byte{0x50, 0x4b, 0x03, 0x04}, 0o644); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}

	if err := service.validateFile(filePath, "sample.docx"); err != nil {
		t.Fatalf("期望小文件校验通过: %v", err)
	}
}

func TestValidateURL(t *testing.T) {
	t.Parallel()

	service := &ConverterService{logger: zerolog.Nop()}

	cases := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{name: "invalid_scheme", url: "file:///etc/passwd", wantErr: true},
		{name: "loopback_ip", url: "http://127.0.0.1/test", wantErr: true},
		{name: "localhost", url: "http://localhost/test", wantErr: true},
		{name: "public_ip", url: "http://8.8.8.8/test", wantErr: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := service.validateURL(context.Background(), tc.url)
			if tc.wantErr && err == nil {
				t.Fatalf("期望返回错误")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("期望校验通过: %v", err)
			}
		})
	}
}
