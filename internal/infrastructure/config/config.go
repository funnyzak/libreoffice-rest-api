package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 统一配置结构。
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Auth      AuthConfig      `mapstructure:"auth"`
	Storage   StorageConfig   `mapstructure:"storage"`
	Worker    WorkerConfig    `mapstructure:"worker"`
	Converter ConverterConfig `mapstructure:"converter"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Logger    LoggerConfig    `mapstructure:"logger"`
	Metrics   MetricsConfig   `mapstructure:"metrics"`
	Swagger   SwaggerConfig   `mapstructure:"swagger"`
	Download  DownloadConfig  `mapstructure:"download"`
	Security  SecurityConfig  `mapstructure:"security"`
	Health    HealthConfig    `mapstructure:"health"`
}

// ServerConfig 服务相关配置。
type ServerConfig struct {
	Host               string `mapstructure:"host"`
	Port               int    `mapstructure:"port"`
	Mode               string `mapstructure:"mode"`
	PublicBaseURL      string `mapstructure:"public_base_url"`
	ReadTimeoutSec     int    `mapstructure:"read_timeout_seconds"`
	WriteTimeoutSec    int    `mapstructure:"write_timeout_seconds"`
	MaxBodyMB          int64  `mapstructure:"max_body_mb"`
	ShutdownTimeoutSec int    `mapstructure:"shutdown_timeout_seconds"`
}

// AuthConfig 认证配置。
type AuthConfig struct {
	Enabled bool     `mapstructure:"enabled"`
	APIKeys []string `mapstructure:"api_keys"`
}

// StorageConfig 存储配置。
type StorageConfig struct {
	TempDir        string `mapstructure:"temp_dir"`
	OutputDir      string `mapstructure:"output_dir"`
	MaxFileMB      int64  `mapstructure:"max_file_mb"`
	RetentionHours int    `mapstructure:"retention_hours"`
}

// WorkerConfig 工作池配置。
type WorkerConfig struct {
	Concurrency int `mapstructure:"concurrency"`
	QueueSize   int `mapstructure:"queue_size"`
}

// ConverterConfig 转换器配置。
type ConverterConfig struct {
	Mode               string    `mapstructure:"mode"`
	TimeoutSec         int       `mapstructure:"timeout_seconds"`
	LibreOfficePath    string    `mapstructure:"libreoffice_path"`
	UserProfileBaseDir string    `mapstructure:"user_profile_base_dir"`
	Uno                UnoConfig `mapstructure:"uno"`
}

// UnoConfig UNO 转换配置。
type UnoConfig struct {
	Enabled                bool   `mapstructure:"enabled"`
	Host                   string `mapstructure:"host"`
	BasePort               int    `mapstructure:"base_port"`
	PoolSize               int    `mapstructure:"pool_size"`
	LibreOfficePath        string `mapstructure:"libreoffice_path"`
	PythonPath             string `mapstructure:"python_path"`
	ScriptPath             string `mapstructure:"script_path"`
	UserProfileBaseDir     string `mapstructure:"user_profile_base_dir"`
	StartupTimeoutSec      int    `mapstructure:"startup_timeout_seconds"`
	ConvertTimeoutSec      int    `mapstructure:"convert_timeout_seconds"`
	RestartAfterJobs       int    `mapstructure:"restart_after_jobs"`
	HealthcheckIntervalSec int    `mapstructure:"healthcheck_interval_seconds"`
}

// DatabaseConfig 数据库配置。
type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

// LoggerConfig 日志配置。
type LoggerConfig struct {
	Level  string           `mapstructure:"level"`
	Format string           `mapstructure:"format"`
	Output string           `mapstructure:"output"`
	File   LoggerFileConfig `mapstructure:"file"`
}

// LoggerFileConfig 文件日志配置。
type LoggerFileConfig struct {
	Enable     bool   `mapstructure:"enable"`
	Path       string `mapstructure:"path"`
	MaxSizeMB  int    `mapstructure:"max_size_mb"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAgeDays int    `mapstructure:"max_age_days"`
	Compress   bool   `mapstructure:"compress"`
}

// MetricsConfig 指标配置。
type MetricsConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	Path        string `mapstructure:"path"`
	RequireAuth bool   `mapstructure:"require_auth"`
}

// SwaggerConfig Swagger 配置。
type SwaggerConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	Path        string `mapstructure:"path"`
	RequireAuth bool   `mapstructure:"require_auth"`
}

// DownloadConfig 下载配置。
type DownloadConfig struct {
	RequireAuth bool `mapstructure:"require_auth"`
}

// SecurityConfig 安全相关配置。
type SecurityConfig struct {
	AllowedMIMETypes  []string `mapstructure:"allowed_mime_types"`
	AllowedExtensions []string `mapstructure:"allowed_extensions"`
	MaxFilenameLength int      `mapstructure:"max_filename_length"`
}

// HealthConfig 健康检查配置。
type HealthConfig struct {
	MinFreeGB int `mapstructure:"min_free_gb"`
}

// Load 从指定路径加载配置。
func Load(path string) (*Config, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("配置文件路径不能为空")
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.SetEnvPrefix("LIBREOFFICE_REST_API")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, fmt.Errorf("配置文件不存在: %s", path)
		}
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 30231)
	v.SetDefault("server.mode", "release")
	v.SetDefault("server.public_base_url", "")
	v.SetDefault("server.read_timeout_seconds", 30)
	v.SetDefault("server.write_timeout_seconds", 30)
	v.SetDefault("server.max_body_mb", 50)
	v.SetDefault("server.shutdown_timeout_seconds", 15)
	v.SetDefault("auth.enabled", true)
	v.SetDefault("storage.temp_dir", "storage/temp")
	v.SetDefault("storage.output_dir", "storage/output")
	v.SetDefault("storage.max_file_mb", 50)
	v.SetDefault("storage.retention_hours", 24)
	v.SetDefault("worker.concurrency", 4)
	v.SetDefault("worker.queue_size", 100)
	v.SetDefault("converter.mode", "cli")
	v.SetDefault("converter.timeout_seconds", 300)
	v.SetDefault("converter.libreoffice_path", "soffice")
	v.SetDefault("converter.user_profile_base_dir", "storage/lo-profile")
	v.SetDefault("converter.uno.enabled", true)
	v.SetDefault("converter.uno.host", "127.0.0.1")
	v.SetDefault("converter.uno.base_port", 2002)
	v.SetDefault("converter.uno.pool_size", 1)
	v.SetDefault("converter.uno.libreoffice_path", "")
	v.SetDefault("converter.uno.python_path", "python3")
	v.SetDefault("converter.uno.script_path", "scripts/uno_convert.py")
	v.SetDefault("converter.uno.user_profile_base_dir", "storage/lo-profile/uno")
	v.SetDefault("converter.uno.startup_timeout_seconds", 15)
	v.SetDefault("converter.uno.convert_timeout_seconds", 120)
	v.SetDefault("converter.uno.restart_after_jobs", 200)
	v.SetDefault("converter.uno.healthcheck_interval_seconds", 10)
	v.SetDefault("database.path", "storage/tasks.db")
	v.SetDefault("logger.level", "info")
	v.SetDefault("logger.format", "json")
	v.SetDefault("logger.output", "stdout")
	v.SetDefault("logger.file.enable", true)
	v.SetDefault("logger.file.path", "logs/app.log")
	v.SetDefault("logger.file.max_size_mb", 50)
	v.SetDefault("logger.file.max_backups", 7)
	v.SetDefault("logger.file.max_age_days", 14)
	v.SetDefault("logger.file.compress", true)
	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.path", "/metrics")
	v.SetDefault("metrics.require_auth", true)
	v.SetDefault("swagger.enabled", true)
	v.SetDefault("swagger.path", "/swagger")
	v.SetDefault("swagger.require_auth", false)
	v.SetDefault("download.require_auth", true)
	v.SetDefault("security.allowed_mime_types", []string{
		"application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.ms-powerpoint",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"application/vnd.oasis.opendocument.text",
		"application/vnd.oasis.opendocument.spreadsheet",
		"application/vnd.oasis.opendocument.presentation",
	})
	v.SetDefault("security.allowed_extensions", []string{".pdf", ".doc", ".docx", ".ppt", ".pptx", ".xls", ".xlsx", ".odt", ".ods", ".odp"})
	v.SetDefault("security.max_filename_length", 128)
	v.SetDefault("health.min_free_gb", 1)
}

// Validate 校验配置合法性。
func (c *Config) Validate() error {
	mode := strings.ToLower(strings.TrimSpace(c.Converter.Mode))
	if mode != "" && mode != "cli" && mode != "uno" {
		return errors.New("转换模式必须为 cli 或 uno")
	}
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return errors.New("端口号范围必须在 1-65535")
	}
	if c.Worker.Concurrency < 1 || c.Worker.Concurrency > 32 {
		return errors.New("工作并发数范围必须在 1-32")
	}
	if c.Converter.TimeoutSec < 5 || c.Converter.TimeoutSec > 600 {
		return errors.New("转换超时时间范围必须在 5-600 秒")
	}
	if mode == "uno" && !c.Converter.Uno.Enabled {
		return errors.New("UNO 模式已启用但 uno.enabled 为 false")
	}
	if mode == "uno" {
		if strings.TrimSpace(c.Converter.Uno.Host) == "" {
			return errors.New("UNO host 不能为空")
		}
		if strings.TrimSpace(c.Converter.Uno.PythonPath) == "" {
			return errors.New("UNO python_path 不能为空")
		}
		if strings.TrimSpace(c.Converter.Uno.ScriptPath) == "" {
			return errors.New("UNO script_path 不能为空")
		}
		if strings.TrimSpace(c.Converter.Uno.UserProfileBaseDir) == "" {
			return errors.New("UNO user_profile_base_dir 不能为空")
		}
		if c.Converter.Uno.BasePort < 1 || c.Converter.Uno.BasePort > 65535 {
			return errors.New("UNO base_port 范围必须在 1-65535")
		}
		if c.Converter.Uno.PoolSize < 1 || c.Converter.Uno.PoolSize > 32 {
			return errors.New("UNO pool_size 范围必须在 1-32")
		}
		if c.Converter.Uno.StartupTimeoutSec < 3 || c.Converter.Uno.StartupTimeoutSec > 120 {
			return errors.New("UNO 启动超时时间范围必须在 3-120 秒")
		}
		if c.Converter.Uno.ConvertTimeoutSec < 5 || c.Converter.Uno.ConvertTimeoutSec > 600 {
			return errors.New("UNO 转换超时时间范围必须在 5-600 秒")
		}
		if c.Converter.Uno.HealthcheckIntervalSec < 3 || c.Converter.Uno.HealthcheckIntervalSec > 300 {
			return errors.New("UNO 健康检查间隔范围必须在 3-300 秒")
		}
		if c.Converter.Uno.RestartAfterJobs < 0 {
			return errors.New("UNO restart_after_jobs 不能为负数")
		}
	}
	if c.Server.ReadTimeoutSec < 5 || c.Server.ReadTimeoutSec > 600 {
		return errors.New("读取超时时间范围必须在 5-600 秒")
	}
	if c.Server.WriteTimeoutSec < 5 || c.Server.WriteTimeoutSec > 600 {
		return errors.New("写入超时时间范围必须在 5-600 秒")
	}
	if strings.TrimSpace(c.Server.PublicBaseURL) != "" {
		parsed, err := url.Parse(strings.TrimSpace(c.Server.PublicBaseURL))
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return errors.New("public_base_url 必须为合法的 http/https URL")
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return errors.New("public_base_url 必须使用 http 或 https 协议")
		}
	}
	if c.Storage.MaxFileMB <= 0 {
		return errors.New("文件大小限制必须大于 0")
	}
	if c.Server.MaxBodyMB <= 0 {
		return errors.New("请求体大小限制必须大于 0")
	}
	if c.Swagger.Enabled && strings.TrimSpace(c.Swagger.Path) == "" {
		return errors.New("Swagger 路径不能为空")
	}
	if c.Storage.RetentionHours <= 0 {
		return errors.New("文件保留时长必须大于 0")
	}
	if c.Security.MaxFilenameLength <= 0 {
		return errors.New("文件名长度限制必须大于 0")
	}
	if c.Auth.Enabled && len(c.Auth.APIKeys) == 0 {
		return errors.New("启用认证时必须配置至少一个 API Key")
	}

	if err := ensureDir(c.Storage.TempDir); err != nil {
		return fmt.Errorf("临时目录不可用: %w", err)
	}
	if err := ensureDir(c.Storage.OutputDir); err != nil {
		return fmt.Errorf("输出目录不可用: %w", err)
	}
	if err := ensureDir(c.Converter.UserProfileBaseDir); err != nil {
		return fmt.Errorf("LibreOffice 用户目录不可用: %w", err)
	}
	if mode == "uno" {
		if err := ensureDir(c.Converter.Uno.UserProfileBaseDir); err != nil {
			return fmt.Errorf("UNO 用户目录不可用: %w", err)
		}
	}
	if err := ensureDir(filepath.Dir(c.Database.Path)); err != nil {
		return fmt.Errorf("数据库目录不可用: %w", err)
	}
	if c.Logger.File.Enable {
		if strings.TrimSpace(c.Logger.File.Path) == "" {
			return errors.New("日志文件路径不能为空")
		}
		if err := ensureDir(filepath.Dir(c.Logger.File.Path)); err != nil {
			return fmt.Errorf("日志目录不可用: %w", err)
		}
	}

	return nil
}

func ensureDir(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("目录不能为空")
	}
	return os.MkdirAll(path, 0o755)
}

// Timeout 返回转换超时配置。
func (c *Config) Timeout() time.Duration {
	return time.Duration(c.Converter.TimeoutSec) * time.Second
}

// UnoConvertTimeout 返回 UNO 转换超时。
func (c *Config) UnoConvertTimeout() time.Duration {
	if c.Converter.Uno.ConvertTimeoutSec <= 0 {
		return c.Timeout()
	}
	return time.Duration(c.Converter.Uno.ConvertTimeoutSec) * time.Second
}

// UnoStartupTimeout 返回 UNO 启动超时。
func (c *Config) UnoStartupTimeout() time.Duration {
	if c.Converter.Uno.StartupTimeoutSec <= 0 {
		return 15 * time.Second
	}
	return time.Duration(c.Converter.Uno.StartupTimeoutSec) * time.Second
}

// UnoHealthcheckInterval 返回 UNO 健康检查间隔。
func (c *Config) UnoHealthcheckInterval() time.Duration {
	if c.Converter.Uno.HealthcheckIntervalSec <= 0 {
		return 10 * time.Second
	}
	return time.Duration(c.Converter.Uno.HealthcheckIntervalSec) * time.Second
}
