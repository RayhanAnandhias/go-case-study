package logger

import (
	"context"
	"log/slog"
	"os"
)

type contextKey string

const TraceIDKey contextKey = "trace_id"

// Custom handler to extract trace_id from context and append to logs
type traceHandler struct {
	slog.Handler
}

func (h *traceHandler) Handle(ctx context.Context, r slog.Record) error {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		r.AddAttrs(slog.String("trace_id", traceID))
	}
	return h.Handler.Handle(ctx, r)
}

func SetupLogger() {
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	
	logger := slog.New(&traceHandler{Handler: jsonHandler})
	slog.SetDefault(logger)
}

func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}
