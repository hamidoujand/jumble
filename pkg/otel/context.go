package otel

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

const defaultTraceID = "00000000000000000000000000000000"

type ctxKey int

const (
	tracerKey  ctxKey = 0
	traceIDKey ctxKey = 1
)

func setTracer(ctx context.Context, tracer trace.Tracer) context.Context {
	return context.WithValue(ctx, tracerKey, tracer)
}

func setTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

func GetTraceID(ctx context.Context) string {
	id, ok := ctx.Value(traceIDKey).(string)
	if !ok {
		return defaultTraceID
	}

	return id
}
