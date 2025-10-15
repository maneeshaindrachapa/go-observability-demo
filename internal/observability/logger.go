package observability

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

func NewLogger() *slog.Logger {
	level := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "debug" {
		level = slog.LevelDebug
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	}))
}

// LogWithTrace adds trace context to logs for correlation
func LogWithTrace(ctx context.Context, logger *slog.Logger, level slog.Level, msg string, args ...any) {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		args = append(args,
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)
	}
	logger.Log(ctx, level, msg, args...)
}

// Helper methods for common log levels
func InfoWithTrace(ctx context.Context, logger *slog.Logger, msg string, args ...any) {
	LogWithTrace(ctx, logger, slog.LevelInfo, msg, args...)
}

func ErrorWithTrace(ctx context.Context, logger *slog.Logger, msg string, args ...any) {
	LogWithTrace(ctx, logger, slog.LevelError, msg, args...)
}

func WarnWithTrace(ctx context.Context, logger *slog.Logger, msg string, args ...any) {
	LogWithTrace(ctx, logger, slog.LevelWarn, msg, args...)
}

func DebugWithTrace(ctx context.Context, logger *slog.Logger, msg string, args ...any) {
	LogWithTrace(ctx, logger, slog.LevelDebug, msg, args...)
}
