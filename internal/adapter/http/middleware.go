package http

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/funnyzak/libreoffice-rest-api/internal/infrastructure/config"
	metrics "github.com/funnyzak/libreoffice-rest-api/internal/infrastructure/metrics"
	domainerrors "github.com/funnyzak/libreoffice-rest-api/pkg/errors"
)

// AuthMiddleware API Key 校验中间件。
func AuthMiddleware(cfg config.AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			apiKey = strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		}
		for _, key := range cfg.APIKeys {
			if key != "" && apiKey == key {
				c.Next()
				return
			}
		}
		WriteError(c, domainerrors.NewAuthentication("认证失败", "API Key 无效", nil))
		c.Abort()
	}
}

// LoggingMiddleware 请求日志中间件。
func LoggingMiddleware(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		method := c.Request.Method
		path := c.Request.URL.Path
		clientIP := c.ClientIP()
		logger.Debug().
			Str("method", method).
			Str("path", path).
			Str("client_ip", clientIP).
			Msg("收到请求")
		c.Next()
		latency := time.Since(start)
		status := c.Writer.Status()
		if routePath := c.FullPath(); routePath != "" {
			path = routePath
		}

		event := logger.Info()
		msg := "请求完成"
		if status >= http.StatusInternalServerError {
			event = logger.Error()
			msg = "请求失败"
		} else if status >= http.StatusBadRequest {
			event = logger.Warn()
			msg = "请求异常"
		}

		event.
			Str("method", method).
			Str("path", path).
			Int("status", status).
			Dur("latency", latency).
			Str("client_ip", clientIP).
			Msg(msg)
	}
}

// SecurityHeadersMiddleware 安全响应头。
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "no-referrer")
		c.Next()
	}
}

// MetricsMiddleware 指标中间件。
func MetricsMiddleware(m *metrics.Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		if m == nil {
			c.Next()
			return
		}
		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()
		m.RequestTotal.WithLabelValues(c.Request.Method, c.FullPath(), http.StatusText(c.Writer.Status())).Inc()
		m.RequestDuration.WithLabelValues(c.Request.Method, c.FullPath()).Observe(duration)
	}
}
