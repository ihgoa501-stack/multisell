package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "multisell_http_requests_total",
			Help: "Total number of HTTP requests by method, path, and status code.",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "multisell_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds by method and path.",
			Buckets: prometheus.DefBuckets, // .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10
		},
		[]string{"method", "path"},
	)

	httpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "multisell_http_requests_in_flight",
			Help: "Current number of HTTP requests being served.",
		},
	)

	httpResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "multisell_http_response_size_bytes",
			Help:    "Response body size in bytes by method and path.",
			Buckets: prometheus.ExponentialBuckets(100, 10, 6), // 100B, 1K, 10K, 100K, 1M, 10M
		},
		[]string{"method", "path"},
	)
)

// Metrics returns a middleware that collects Prometheus HTTP metrics.
// Uses Gin's FullPath() to group by route pattern, keeping cardinality bounded.
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		httpRequestsInFlight.Inc()
		start := time.Now()

		// Use the route pattern to keep cardinality bounded.
		// e.g. /api/v1/order/:id instead of /api/v1/order/123
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		c.Next()

		duration := time.Since(start)
		status := strconv.Itoa(c.Writer.Status())

		httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration.Seconds())
		httpResponseSize.WithLabelValues(c.Request.Method, path).Observe(float64(c.Writer.Size()))
		httpRequestsInFlight.Dec()
	}
}

// MetricsHandler returns a Gin handler that serves Prometheus /metrics output.
func MetricsHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
