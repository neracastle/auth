package tracer

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

var globalTracer trace.Tracer

func Init(t trace.Tracer) {
	globalTracer = t
}

func Tracer() trace.Tracer {
	return globalTracer
}
func Span(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return globalTracer.Start(ctx, spanName, opts...)
}
