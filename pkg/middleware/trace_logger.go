package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go-case-study/pkg/logger"
)

func TraceLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := uuid.New().String()
		ctx := context.WithValue(c.Request.Context(), logger.TraceIDKey, traceID)
		c.Request = c.Request.WithContext(ctx)

		slog.InfoContext(ctx, "request started",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
		)

		start := time.Now()
		c.Next()
		duration := time.Since(start)

		slog.InfoContext(ctx, "request completed",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", duration.Milliseconds(),
		)
	}
}
