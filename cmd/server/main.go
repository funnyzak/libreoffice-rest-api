package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	adapter "github.com/funnyzak/libreoffice-rest-api/internal/adapter/http"
	"github.com/funnyzak/libreoffice-rest-api/internal/core/libreoffice"
	"github.com/funnyzak/libreoffice-rest-api/internal/core/workerpool"
	"github.com/funnyzak/libreoffice-rest-api/internal/infrastructure/cleaner"
	"github.com/funnyzak/libreoffice-rest-api/internal/infrastructure/config"
	"github.com/funnyzak/libreoffice-rest-api/internal/infrastructure/logger"
	metrics "github.com/funnyzak/libreoffice-rest-api/internal/infrastructure/metrics"
	"github.com/funnyzak/libreoffice-rest-api/internal/repository"
	"github.com/funnyzak/libreoffice-rest-api/internal/service"
)

// @title libreoffice-rest-api
// @version 1.0.0
// @description LibreOffice 文档转换 REST API
// @host localhost:30231
// @BasePath /
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
func main() {
	rootCmd := &cobra.Command{
		Use:   "libreoffice-rest-api",
		Short: "LibreOffice 文档转换服务",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, _ := cmd.Flags().GetString("config")
			return runServer(cfgPath)
		},
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "显示版本信息",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Version: %s\nCommit: %s\nBuild Date: %s\n", version, commit, buildDate)
		},
	}

	rootCmd.Flags().String("config", "config.yaml", "配置文件路径")
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runServer(cfgPath string) error {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}

	log, err := logger.Init(cfg.Logger)
	if err != nil {
		return err
	}
	log.Info().
		Str("version", version).
		Str("commit", commit).
		Str("build_date", buildDate).
		Str("config_path", cfgPath).
		Str("mode", cfg.Server.Mode).
		Int("port", cfg.Server.Port).
		Int("worker_concurrency", cfg.Worker.Concurrency).
		Int("worker_queue_size", cfg.Worker.QueueSize).
		Str("output_dir", cfg.Storage.OutputDir).
		Str("temp_dir", cfg.Storage.TempDir).
		Str("libreoffice_path", cfg.Converter.LibreOfficePath).
		Int("api_key_count", len(cfg.Auth.APIKeys)).
		Bool("auth_enabled", cfg.Auth.Enabled).
		Str("log_level", cfg.Logger.Level).
		Str("log_format", cfg.Logger.Format).
		Str("log_output", cfg.Logger.Output).
		Msg("配置加载完成")

	repo, err := repository.NewSQLiteRepository(cfg.Database.Path)
	if err != nil {
		return err
	}

	pool := workerpool.New(cfg.Worker.Concurrency, cfg.Worker.QueueSize, log)
	pool.Start()

	executor := &libreoffice.CommandExecutor{
		LibreOfficePath:    cfg.Converter.LibreOfficePath,
		UserProfileBaseDir: cfg.Converter.UserProfileBaseDir,
		Timeout:            cfg.Timeout(),
	}

	metricsCollector := (*metrics.Metrics)(nil)
	if cfg.Metrics.Enabled {
		metricsCollector = metrics.NewMetrics()
	}

	converter := service.NewConverterService(repo, executor, pool, cfg, log)
	tasks := service.NewTaskService(repo, log)
	handler := adapter.NewHandler(converter, tasks, repo, cfg, log)

	router := adapter.NewRouter(cfg)
	adapter.RegisterRoutes(router, handler, cfg, metricsCollector)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutSec) * time.Second,
	}

	cleanerCtx, cleanerCancel := context.WithCancel(context.Background())
	defer cleanerCancel()
	cleanerWorker := cleaner.New(repo, time.Hour, log)
	go cleanerWorker.Start(cleanerCtx)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Info().Str("addr", srv.Addr).Msg("服务启动")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("服务启动失败")
			stop()
		}
	}()

	<-ctx.Done()
	log.Info().Msg("收到关闭信号")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Server.ShutdownTimeoutSec)*time.Second)
	defer cancel()

	cleanerCancel()
	_ = pool.Shutdown(shutdownCtx)
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("服务关闭失败")
	}

	log.Info().Msg("服务已停止")
	return nil
}
