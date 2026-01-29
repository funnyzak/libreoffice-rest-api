package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	docs "github.com/funnyzak/libreoffice-rest-api/docs"
	"github.com/funnyzak/libreoffice-rest-api/internal/infrastructure/config"
	metrics "github.com/funnyzak/libreoffice-rest-api/internal/infrastructure/metrics"
)

// RegisterRoutes 注册路由。
func RegisterRoutes(router *gin.Engine, handler *Handler, cfg *config.Config, metricsCollector *metrics.Metrics) {
	router.Use(LoggingMiddleware(handler.logger))
	router.Use(SecurityHeadersMiddleware())
	router.Use(MetricsMiddleware(metricsCollector))

	if cfg.Swagger.Enabled {
		docs.SwaggerInfo.BasePath = "/"
		if cfg.Swagger.RequireAuth {
			router.GET(swaggerPattern(cfg.Swagger.Path), AuthMiddleware(cfg.Auth), ginSwagger.WrapHandler(swaggerFiles.Handler))
		} else {
			router.GET(swaggerPattern(cfg.Swagger.Path), ginSwagger.WrapHandler(swaggerFiles.Handler))
		}
	}

	api := router.Group("/api/v1")
	api.Use(AuthMiddleware(cfg.Auth))
	{
		api.POST("/convert", handler.Convert)
		api.POST("/merge", handler.Merge)
		api.GET("/tasks/:id", handler.GetTask)
		if cfg.Download.RequireAuth {
			api.GET("/files/:id/download", handler.Download)
		}
	}
	if !cfg.Download.RequireAuth {
		router.GET("/api/v1/files/:id/download", handler.Download)
	}

	router.GET("/health", handler.Health)

	if cfg.Metrics.Enabled {
		if cfg.Metrics.RequireAuth {
			router.GET(cfg.Metrics.Path, AuthMiddleware(cfg.Auth), gin.WrapH(metrics.PrometheusHandler()))
		} else {
			router.GET(cfg.Metrics.Path, gin.WrapH(metrics.PrometheusHandler()))
		}
	}
}

// NewRouter 创建路由。
func NewRouter(cfg *config.Config) *gin.Engine {
	gin.SetMode(cfg.Server.Mode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.MaxMultipartMemory = cfg.Storage.MaxFileMB * 1024 * 1024
	router.NoRoute(notFound)
	return router
}

func notFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, APIResponse{
		Success: false,
		Error: &ErrorDetail{
			Code:    http.StatusNotFound,
			Message: "资源不存在",
			Details: "请求路径未匹配",
		},
	})
}

func swaggerPattern(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/swagger/*any"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimRight(path, "/") + "/*any"
}
