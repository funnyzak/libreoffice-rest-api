package logger

import (
	"errors"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/funnyzak/libreoffice-rest-api/internal/infrastructure/config"
)

// Init 初始化日志器。
func Init(cfg config.LoggerConfig) (zerolog.Logger, error) {
	level, err := zerolog.ParseLevel(strings.ToLower(cfg.Level))
	if err != nil {
		return zerolog.Logger{}, errors.New("日志级别不合法")
	}

	writers := make([]io.Writer, 0, 2)

	if cfg.Output == "stdout" || cfg.Output == "both" {
		writers = append(writers, os.Stdout)
	}
	if cfg.Output == "file" || cfg.Output == "both" {
		if !cfg.File.Enable {
			return zerolog.Logger{}, errors.New("文件日志未启用")
		}
		writers = append(writers, newFileWriter(cfg.File))
	}
	if len(writers) == 0 {
		return zerolog.Logger{}, errors.New("日志输出方式不合法")
	}

	output := io.MultiWriter(writers...)
	if strings.ToLower(cfg.Format) == "console" {
		output = zerolog.ConsoleWriter{Out: output, TimeFormat: time.RFC3339}
	}

	logger := zerolog.New(output).With().Timestamp().Logger().Level(level)
	zerolog.TimeFieldFormat = time.RFC3339

	return logger, nil
}

func newFileWriter(cfg config.LoggerFileConfig) io.Writer {
	return &lumberjack.Logger{
		Filename:   cfg.Path,
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
		Compress:   cfg.Compress,
	}
}
