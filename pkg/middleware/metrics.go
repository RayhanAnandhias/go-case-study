package middleware

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"service", "endpoint", "status_code"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "endpoint"},
	)
)

// PrometheusMetrics is a Gin middleware to record HTTP metrics
func PrometheusMetrics(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Continue to the next middleware or handler
		c.Next()

		// Skip measuring for /metrics or /health
		if strings.Contains(c.Request.URL.Path, "/metrics") ||
			strings.Contains(c.Request.URL.Path, "/health") {
			return
		}

		duration := time.Since(start).Seconds()
		statusCode := strconv.Itoa(c.Writer.Status())
		endpoint := c.FullPath()
		if endpoint == "" {
			endpoint = c.Request.URL.Path
		}

		httpRequestsTotal.WithLabelValues(serviceName, endpoint, statusCode).Inc()
		httpRequestDuration.WithLabelValues(serviceName, endpoint).Observe(duration)
	}
}
