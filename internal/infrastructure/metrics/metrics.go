package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics 指标集合。
type Metrics struct {
	RequestTotal    *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	TaskTotal       *prometheus.CounterVec
	ActiveTasks     prometheus.Gauge
}

// NewMetrics 创建指标实例。
func NewMetrics() *Metrics {
	m := &Metrics{
		RequestTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "libreoffice",
				Subsystem: "http",
				Name:      "request_total",
				Help:      "HTTP 请求总数",
			},
			[]string{"method", "path", "status"},
		),
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "libreoffice",
				Subsystem: "http",
				Name:      "request_duration_seconds",
				Help:      "HTTP 请求耗时",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		TaskTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "libreoffice",
				Subsystem: "task",
				Name:      "total",
				Help:      "任务总数",
			},
			[]string{"status"},
		),
		ActiveTasks: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "libreoffice",
				Subsystem: "task",
				Name:      "active",
				Help:      "活跃任务数",
			},
		),
	}
	prometheus.MustRegister(m.RequestTotal, m.RequestDuration, m.TaskTotal, m.ActiveTasks)
	return m
}

// PrometheusHandler 返回指标 Handler。
func PrometheusHandler() http.Handler {
	return promhttp.Handler()
}
