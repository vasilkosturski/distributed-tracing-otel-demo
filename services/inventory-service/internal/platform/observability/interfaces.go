package observability

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Logger defines the interface for application logging.
// This is a platform concern that deals with external logging systems.
type Logger interface {
	Info(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	With(fields ...zap.Field) *zap.Logger
	Sync() error
}

// Tracer defines the interface for distributed tracing.
// This is a platform concern that deals with external tracing systems.
type Tracer interface {
	Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span)
}
